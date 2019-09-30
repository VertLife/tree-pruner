package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	mailgun "github.com/mailgun/mailgun-go/v3"
	log "github.com/sirupsen/logrus"
	// "google.golang.org/appengine"
)

// type JobConstants struct {
// 	MinSampleSize, MaxTreeSize                              int
// 	TreeFilePath, OutputBasePath, ConfigPath, DefaultBucket string
// }

// Loadable for translating form values to struct
// https://stackoverflow.com/a/12905824
type Loadable interface {
	LoadValue(name string, value []string)
}

// LoadModel for translating form values to struct
func LoadModel(loadable Loadable, data map[string][]string) {
	for key, value := range data {
		loadable.LoadValue(key, value)
	}
}

// JobDetail represets a single job request
type JobDetail struct {
	JobID       string              `yaml:"job_id"`
	Email       string              `yaml:"email"`
	BaseTree    string              `yaml:"tree_base"`
	TreeSet     string              `yaml:"tree_set"`
	SampleSize  int                 `yaml:"sample_size"`
	Species     []string            `yaml:"names,flow"`
	SampleTrees []string            `yaml:"sample_trees,flow"`
	Status      string              `yaml:"status"`
	CreatedAt   string              `yaml:"created_at"`
	CompletedAt string              `yaml:"completed_at"`
	BadNames    map[string][]string `yaml:"bad_names,flow"`
}

// LoadValue is overloaded
func (jd *JobDetail) LoadValue(name string, value []string) {
	switch name {
	case "email":
		jd.Email = cleanup(value[0])[0]
	case "tree_base":
		jd.BaseTree = cleanup(value[0])[0]
	case "tree_set":
		jd.TreeSet = cleanup(value[0])[0]
	case "species":
		jd.Species = cleanup(value[0])
	case "sample_size":
		i, err := strconv.Atoi(cleanup(value[0])[0])
		if err != nil {
			jd.SampleSize = MinSampleSize
		} else {
			jd.SampleSize = i
		}
	}
}

// SendEmail for users
func (jd *JobDetail) SendEmail(ctx context.Context, subject, body string) {

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_API_KEY"))

	mm := mg.NewMessage(
		os.Getenv("MAILGUN_SENDER"),
		subject,
		body,
		jd.Email)
	mm.AddHeader("Sender", os.Getenv("MAILGUN_SENDER"))

	ctxd, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	msg, id, err := mg.Send(ctxd, mm)
	if err != nil {
		log.Error("Could not send message: ", err, ", ID ", id, ", ", msg)
		return
	}
}

// SendJobCreatedEmail is a helper function to send the initial email
func (jd *JobDetail) SendJobCreatedEmail(ctx context.Context) {
	subj := fmt.Sprintf("[%s] Your request is being processed", TreeSiteCodes[jd.BaseTree])
	msg := fmt.Sprintf(`Thank you for using the %s service to generate your phylogeny subsets.

	Your request has been received and is currently being processed. You will recieve a confirmation email when your results have been completed.

        Please visit http://%s/subsets/ to check the status of your request, using the following information:
          Task ID: %s
		  Email: %s
				
	Results taking longer than expected?
	Traffic volume may result in longer processing times. If you don't recieve an email with the completed subset trees within 24 hours, please contact us at support@vertlife.org
    `, TreeSiteCodes[jd.BaseTree], TreeSiteUrls[jd.BaseTree], jd.JobID, jd.Email)
	jd.SendEmail(ctx, subj, msg)
}

// SendJobDoneEmail is a helper function to send the final email
func (jd *JobDetail) SendJobDoneEmail(ctx context.Context) {
	subj := fmt.Sprintf("[%s] Your pruned trees are ready", TreeSiteCodes[jd.BaseTree])
	msg := fmt.Sprintf(`Thank you for using the %s service to generate your phylogeny subsets.

        You can access your tree results and additional information here:
          Pruned Trees: %s
          Task ID: %s
    `, TreeSiteCodes[jd.BaseTree], "https://data.vertlife.org/pruned_treesets/"+jd.JobID+"/"+jd.JobID+".zip", jd.JobID)
	jd.SendEmail(ctx, subj, msg)
}

// SendJobFailedEmail is a helper function to send the failed email
func (jd *JobDetail) SendJobFailedEmail(ctx context.Context) {
	subj := fmt.Sprintf("[%s] Issue with your request", TreeSiteCodes[jd.BaseTree])
	msg := fmt.Sprintf(`There was an issue with your %s service while generating your phylogeny subsets.
		
        Please try again, and if the problem persists, please contact us at support@vertlife.org with the following information:
          Task ID: %s
          Email: %s
    `, TreeSiteCodes[jd.BaseTree], jd.JobID, jd.Email)
	jd.SendEmail(ctx, subj, msg)
}

// SendJobUsage tracks the job request
func (jd *JobDetail) SendJobUsage(ctx context.Context) {
	// ctxd, cancel := context.WithTimeout(ctx, 60*time.Second)
	// defer cancel()

	qstmpl := "INSERT INTO usage (email, job_id, basetree, treeset, sample_size) VALUES ( '%s', '%s', '%s', '%s', %d )"
	qs := fmt.Sprintf(qstmpl, jd.Email, jd.JobID, jd.BaseTree, jd.TreeSet, jd.SampleSize)

	v := url.Values{}
	v.Set("q", qs)
	v.Add("api_key", os.Getenv("CARTO_API_KEY"))

	_, err := http.Get(os.Getenv("CARTO_URL") + "?" + v.Encode())
	if err != nil {
		log.Error("Could not add job to usage list:ID ", jd.JobID, ", ", err)
	}
	// body, err := ioutil.ReadAll(res.Body)
	// res.Body.Close()
	// if err != nil {
	// 	log.Error("Could not read usage response:ID %v, %v", jd.JobID, err)
	// }
	log.Info("Added job to usage: ", jd.JobID)
}
