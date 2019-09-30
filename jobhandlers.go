package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/evolbioinfo/gotree/io/newick"
	"github.com/evolbioinfo/gotree/tree"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func InitPruneJob(ctx context.Context, jobID string, formData map[string][]string) error {

	// Check email
	if len(cleanup(formData["email"][0])) == 0 {
		return errors.New("Please provide a valid email")
	}

	// Check valid base tree
	if len(cleanup(formData["tree_base"][0])) == 0 || !HasKey(TreeSiteCodes, cleanup(formData["tree_base"][0])[0]) {
		return errors.New("Please provide a valid base tree")
	}

	// Check valid treeset
	if len(cleanup(formData["tree_set"][0])) == 0 || !HasKey(TreeCodes, cleanup(formData["tree_set"][0])[0]) {
		return errors.New("Please provide a valid tree source")
	}

	// Check valid sample size
	if len(cleanup(formData["sample_size"][0])) == 0 {
		return errors.New("Please provide a valid number between 100 - 10000")
	}
	ss, err := strconv.Atoi(cleanup(formData["sample_size"][0])[0])
	if err != nil {
		return errors.New("Please provide a valid number between 100 - 10000")
	}
	if ss < MinSampleSize || ss > MaxTreeSize {
		return errors.New("Please provide a valid number between 100 - 10000")
	}

	// Check valid species list
	if len(cleanup(formData["species"][0])) == 0 {
		return errors.New("Please provide a valid set of species to prune")
	}

	jd := new(JobDetail)
	LoadModel(jd, formData)

	jd.JobID = jobID
	jd.Status = "CREATED"
	jd.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Generate the necessary sample trees and write to config
	currentMaxTreeSize := MaxTreeSize
	if jd.BaseTree == "sharktree" && strings.Index(jd.TreeSet, "sequence_data") > 0 {
		currentMaxTreeSize = 500
	}
	rand.Seed(time.Now().UnixNano())
	samples := rand.Perm(currentMaxTreeSize)[:jd.SampleSize]
	sampleTrees := make([]string, jd.SampleSize)
	for ti, treenum := range samples {
		sampleTrees[ti] = fmt.Sprintf("%s_%04d.tre", jd.TreeSet, treenum)
	}
	jd.SampleTrees = sampleTrees

	// jds, err := yaml.Marshal(&jd)
	// if err != nil {
	// 	log.Error("Error setting up job details: %v", err)
	// }
	// log.Debug("\n%s", string(jds))

	// Write the job config file
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error("failed to create client: ", err)
		return errors.New("Unable to setup config")
	}
	defer client.Close()

	bkt := client.Bucket(DefaultBucket)
	err = writeJobDetail(ctx, bkt, jd)
	if err != nil {
		log.Error("Unable to write config: ", err)
		emsg := fmt.Sprintf("Unable to write config: %v", err)
		return NewError(5, jd.JobID, emsg, err)
	}

	jd.SendJobCreatedEmail(ctx)
	jd.SendJobUsage(ctx)

	// log.Debug("\n=============================\n")

	return nil
}
func DummyJob(ctx context.Context, jobID string) {
	log.Debug("\n\nDoing nothing\n\n")

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error("failed to create client: ", err)
		return
	}
	defer client.Close()

	bkt := client.Bucket(DefaultBucket)

	jd, err := loadJobDetails(ctx, bkt, jobID)
	if err != nil {
		log.Error("Error loading job details\n ", err)
		return
	}

	log.Debug("\n=============================\n")
	log.Debug("\nCONFIG TO LOAD\n")
	log.Debug("\n", jd)
	log.Debug("\n=============================\n")
	log.Debug("\n Hello ", jd.Email)
	log.Debug("\n=============================\n")

	rand.Seed(time.Now().UnixNano())
	samples := rand.Perm(10000)[:5]

	log.Debug("=================")
	log.Debug("samples")
	log.Debug("=================")
	log.Debug(samples)
	log.Debug("=================")

}

func loadJobDetails(ctx context.Context, bkt *storage.BucketHandle, jobID string) (*JobDetail, error) {

	cfgfile := fmt.Sprintf(ConfigPath, jobID)
	rc, err := bkt.Object(cfgfile).NewReader(ctx)
	if err != nil {
		log.Error("readFile: unable to open config file: ", cfgfile, "\n", err)
		return nil, err
	}
	defer rc.Close()

	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		log.Error("readFile: unable to read data from config file ", cfgfile, ": ", err)
		return nil, err
	}

	jd := &JobDetail{}
	err = yaml.Unmarshal(slurp, jd)
	if err != nil {
		log.Error("readFile: unable to marshall config: ", err)
		return nil, err
	}

	return jd, nil
}

func writeJobDetail(ctx context.Context, bkt *storage.BucketHandle, jd *JobDetail) error {

	cfgfile := fmt.Sprintf(ConfigPath, jd.JobID)
	wc := bkt.Object(cfgfile).NewWriter(ctx)

	wc.ContentType = "text/plain"

	jds, err := yaml.Marshal(jd)
	if err != nil {
		return err
	}

	if _, err := wc.Write(jds); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

func StartPruneJob(ctx context.Context, jobID string) (*JobDetail, error) {

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error("failed to create client: ", err)
		// jd.SendJobFailedEmail(ctx)
		// return
		emsg := fmt.Sprintf("failed to create client: %v", err)
		return nil, NewError(1, jobID, emsg, err)
	}
	defer client.Close()

	bkt := client.Bucket(DefaultBucket)

	jd, err := loadJobDetails(ctx, bkt, jobID)
	if err != nil {
		log.Error("error loading job details\n %v", err)
		emsg := fmt.Sprintf("Error loading job details\n %v", err)
		return nil, NewError(0, jobID, emsg, err)
	}

	namesToKeep := jd.Species
	for mi := range namesToKeep {
		namesToKeep[mi] = strings.Replace(namesToKeep[mi], " ", "_", -1)
	}

	badNamesList := make(map[string][]string)
	for ti, st := range jd.SampleTrees {
		treefile := fmt.Sprintf(TreeFilePath, jd.BaseTree, st)
		tobj := bkt.Object(treefile)
		outfile := fmt.Sprintf(OutputBasePath, jobID, st)

		// Check if we've already pruned this tree for the current job
		// This happens if the task was restarted
		prc, err := bkt.Object(outfile).NewReader(ctx)
		if err == nil {
			// NO ERROR
			// Means, tree has been pruned and stored. Let's skip it
			// log.Debug("Skipping already done tree: %d = %q\n\n", ti, treefile)
			prc.Close()
			continue
		}

		rc, err := tobj.NewReader(ctx)
		if err != nil {
			log.Error("Unable to open tree file: ", treefile, "\n ", err)
			// jd.SendJobFailedEmail(ctx)
			// return
			emsg := fmt.Sprintf("Unable to open tree file: %q\n %v", treefile, err)
			return jd, NewError(2, jd.JobID, emsg, err)
		}
		// defer rc.Close()

		log.Info("Will prune tree: ", ti, " = ", treefile)
		ptree, badNames, err := startPrune(ctx, rc, namesToKeep)
		if err != nil {
			log.Debug("Oh noes. We have the errors: ", err)
		}

		if err := rc.Close(); err != nil {
			log.Error("Unable to close tree file: ", treefile, "\n ", err)
		}

		// log.Debug("Will write to: %q\n\n", outfile)
		badNamesList[st] = badNames

		// writeFile(outfile, ptree)
		wc := bkt.Object(outfile).NewWriter(ctx)
		wc.ContentType = "text/plain"

		if _, err := wc.Write([]byte(ptree)); err != nil {
			log.Error("Unable to write pruned tree file, ", outfile, ": ", err)
			// jd.SendJobFailedEmail(ctx)
			// return
			emsg := fmt.Sprintf("Unable to write pruned tree file, %q: %v", outfile, err)
			return jd, NewError(3, jd.JobID, emsg, err)
		}
		if err := wc.Close(); err != nil {
			log.Error("Unable to close pruned tree file, ", outfile, ": ", err)
		}
	}

	// Update the JobDetail with the bad names
	jd.BadNames = badNamesList

	ferr := finaliseJob(ctx, jd, bkt)
	return jd, ferr
}

func getHeader(jd *JobDetail) string {
	citLong := TreeCitationLong[jd.BaseTree]
	citShort := TreeCitationShort[jd.BaseTree]
	siteURL := TreeSiteUrls[jd.BaseTree]
	treeset := TreeCodes[jd.TreeSet]
	createdAt := jd.CreatedAt

	return fmt.Sprintf(`#NEXUS

[Tree distribution from: "%s]
[Subsampled and pruned from %s on %s ]
[Data: "%s" (see %s supplement for details)]

BEGIN TREES;`, citLong, siteURL, createdAt, treeset, citShort)
}

func readFile(ctx context.Context, bkt *storage.BucketHandle, filename string) ([]byte, error) {
	rc, err := bkt.Object(filename).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func finaliseJob(ctx context.Context, jd *JobDetail, bkt *storage.BucketHandle) error {
	log.Info("Finalising job: ", jd.JobID)

	outputDir := fmt.Sprintf(OutputBaseDir, jd.JobID)
	outputTreeFile := outputDir + "output.nex"

	owc := bkt.Object(outputTreeFile).NewWriter(ctx)
	// defer owc.Close()  // Explicitly closing this later!

	// Write the header
	owc.Write([]byte(getHeader(jd)))

	re := regexp.MustCompile("[0-9]+")
	for _, st := range jd.SampleTrees {

		// Get the pruned tree data
		prunedFile := fmt.Sprintf(OutputBasePath, jd.JobID, st)

		// Get the tree number
		// TODO: Add a check here to make sure we do have a number!
		//       And that it's the last number!
		tnums := re.FindAllString(prunedFile, -1)
		treenum := tnums[len(tnums)-1]

		data, err := readFile(ctx, bkt, prunedFile)
		if err != nil {
			log.Error("Unable to read pruned tree file, ", prunedFile, ": ", err)

			// Let's continue for now.
			// TODO: We should catch an error and email the user. Or Support!
			owc.Write([]byte("\n\tTREE tree_" + treenum + " = (NO TREE FILE. FIX THIS.);"))
			continue
		}

		owc.Write([]byte("\n\tTREE tree_" + treenum + " = "))
		if _, err := owc.Write(data); err != nil {
			log.Error("Unable to concat pruned tree file, ", prunedFile, ": ", err)

			// Let's continue for now.
			// TODO: We should catch an error and email the user. Or Support!
			owc.Write([]byte("\n\tTREE tree_" + treenum + " = (NO TREE FILE. FIX THIS.);"))
			continue
		}
	}

	// Write the END tag
	owc.Write([]byte("\nEND;\n\n"))
	if err := owc.Close(); err != nil {
		log.Error("Unable to close output file ", outputTreeFile, ": ", err)
		// jd.SendJobFailedEmail(ctx)
		// return
		emsg := fmt.Sprintf("Unable to close output file %q: %v", outputTreeFile, err)
		return NewError(4, jd.JobID, emsg, err)
	}

	// Update the job status
	jd.Status = "COMPLETED"
	jd.CompletedAt = time.Now().UTC().Format(time.RFC3339)

	// Dump the job file to storage
	err := writeJobDetail(ctx, bkt, jd)
	if err != nil {
		log.Error("Unable to write config: ", err)
		emsg := fmt.Sprintf("Unable to write config: %v", err)
		return NewError(5, jd.JobID, emsg, err)
	}

	if err = zipFiles(ctx, jd, bkt); err != nil {
		log.Error("Error zipping data: ", err)
		// jd.SendJobFailedEmail(ctx)
		emsg := fmt.Sprintf("Error zipping data: %v", err)
		return NewError(6, jd.JobID, emsg, err)
	}

	jd.SendJobDoneEmail(ctx)

	log.Info("Completed job: ", jd.JobID)
	return nil
}

func zipFiles(ctx context.Context, jd *JobDetail, bkt *storage.BucketHandle) error {

	outputDir := fmt.Sprintf(OutputBaseDir, jd.JobID)

	zipFilePath := outputDir + jd.JobID + ".zip"
	zobj := bkt.Object(zipFilePath)
	zwc := zobj.NewWriter(ctx)

	// Zip up the files for email
	zw := zip.NewWriter(zwc)
	for _, file := range []string{"output.nex", "config.yaml"} {

		filename := outputDir + file
		// log.Info("File filename: \n%v\n", filename)

		fobj := bkt.Object(filename)
		objAttrs, err := fobj.Attrs(ctx)

		zf, err := zw.Create(file)
		if err != nil {
			log.Error("Unable to add output to zip file: ", err)
			continue
		}

		var currentByte int64 = 0
		var byteSize int64 = 5 * 1024 * 1024
		buf := make([]byte, byteSize)
		for {
			bytesToRead := byteSize
			if currentByte+byteSize > objAttrs.Size {
				bytesToRead = objAttrs.Size - currentByte
			}
			rc, err := fobj.NewRangeReader(ctx, currentByte, bytesToRead)
			if err != nil {
				log.Error("Failed opening file: ", err)
				return err
			}

			if _, err := io.CopyBuffer(zf, rc, buf); err != nil {
				log.Error("Failed copying file: ", err)
				return err
			}
			rc.Close()

			if bytesToRead < byteSize {
				break
			}

			currentByte += byteSize
		}
	}

	// Close the zip buffer
	if err := zw.Close(); err != nil {
		log.Error("Unable to create zip file: ", err)
		return err
	}

	if err := zwc.Close(); err != nil {
		log.Error("Unable to create zip file: ", err)
		return err
	}

	// Set the zip permissions
	// log.Info("Setting zip permissions")
	ferr := zobj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader)
	if ferr != nil {
		log.Error("Unable to set permissions on zip file: ", ferr)
		return ferr
	}
	// log.Info("Done setting zip permissions")

	return nil
}

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func HasKey(vs map[string]string, t string) bool {
	for k := range vs {
		if k == t {
			return true
		}
	}
	return false
}

func writeFile(filename, treedata string) {
	var f *os.File
	var err error

	err = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		panic(err)
	}

	if f, err = os.Create(filename); err != nil {
		panic(err)
	}
	defer f.Close()

	f.WriteString(treedata)
	f.Sync()
}

func startPrune(ctx context.Context, reader *storage.Reader, namesToKeep []string) (string, []string, error) {
	var t *tree.Tree
	var err error

	t, err = newick.NewParser(reader).Parse()
	if err != nil {
		return "", nil, err
	}

	allNames := t.AllTipNames()
	badNames := make([]string, 0)

	for _, mn := range namesToKeep {
		if Index(allNames, mn) == -1 {
			badNames = append(badNames, mn)
		}
	}

	err = t.RemoveTips(true, namesToKeep...)
	if err != nil {
		return "", nil, err
	}

	return t.Newick(), badNames, nil
}

func cleanup(fv string) []string {
	a := strings.Split(fv, "\n")
	b := a[:0]
	for _, x := range a {
		if x != "" {
			b = append(b, strings.TrimSpace(x))
		}
	}
	return b
}
