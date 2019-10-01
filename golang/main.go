package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/storage"
	uuid "github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/api/prune", start)
	http.HandleFunc("/api/result", result)
	http.HandleFunc("/api/prune/worker", worker)
	http.HandleFunc("/api/test", test)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Hello, World!")
}

func test(w http.ResponseWriter, r *http.Request) {
	// ctx := context.Background()

	// log.Debug("Is Dev Server: %v", appengine.IsDevAppServer())

	log.Debug("Calling error")

	err := NewError(1, "job-id-here", "what is this error even?", nil)
	log.Error(err)

	err2 := NewError(1, "", "No job id.", nil)
	// log.Error("%v", err2)
	log.Error(err2)

	log.Debug("Done with error")
}

func start(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	juuid, err := uuid.NewV4()
	if err != nil {
		log.Error("Error generating job id: ", err)
	}

	if err := r.ParseForm(); err != nil {
		log.Error("Errror parsing form: ", err)
	}

	jobID := fmt.Sprintf("tree-pruner-%s", juuid)

	// log.Debug("form: \n %d - %v", len(r.PostForm), r.PostForm)

	if len(r.PostForm) == 0 {
		w.WriteHeader(400)
		w.Write([]byte(`{"status": "ERROR", "message": "Please provide all the necessary data"}`))
		return
	}

	err = InitPruneJob(ctx, jobID, r.PostForm)
	if err != nil {
		// http.Error(w, err.Error(), http.StatusInternalServerError)
		resData, _ := json.Marshal(&JobResult{
			JobID:   jobID,
			Status:  "ERROR",
			Message: err.Error(),
		})

		w.WriteHeader(400)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(resData)
		return
	}

	// t := taskqueue.NewPOSTTask("/api/prune/worker", map[string][]string{"job_id": {jobID}})
	// if _, err := taskqueue.Add(ctx, t, DefaultQueue); err != nil {
	// 	// http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	resData, _ := json.Marshal(&JobResult{
	// 		JobID:   jobID,
	// 		Status:  "ERROR",
	// 		Message: err.Error(),
	// 	})

	// 	w.WriteHeader(400)
	// 	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.Write(resData)
	// 	return
	// }

	projectID := os.Getenv("TASK_PROJECT")
	locationID := os.Getenv("TASK_LOCATION")
	queueID := os.Getenv("TASK_QUEUE")
	taskRoute := os.Getenv("TASK_ROUTE")
	taskService := os.Getenv("TASK_SERVICE")
	taskVersion := os.Getenv("TASK_VERSION")

	ctxd := context.Background()
	client, err := cloudtasks.NewClient(ctxd)
	if err != nil {
		// return nil, fmt.Errorf("NewClient: %v", err)
		log.Error("Error creating cloud tasks client: ", err)
		return
	}

	// Build the Task queue path.
	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	// Build the Task payload.
	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#AppEngineHttpRequest
			MessageType: &taskspb.Task_AppEngineHttpRequest{
				AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
					HttpMethod:  taskspb.HttpMethod_POST,
					RelativeUri: taskRoute,
					AppEngineRouting: &taskspb.AppEngineRouting{
						Service: taskService,
						Version: taskVersion,
					},
				},
			},
		},
	}

	// Add a payload message if one is present.
	req.Task.GetAppEngineHttpRequest().Body = []byte(jobID)

	createdTask, err := client.CreateTask(ctx, req)
	if err != nil {
		// return nil, fmt.Errorf("cloudtasks.CreateTask: %v", err)
		log.Error("Error creating cloud task: ", err)
		return
	}
	log.Info("Create Task: ", createdTask.GetName())

	log.Info("Started prune job: ", jobID)

	resData, _ := json.Marshal(&JobResult{
		JobID:   jobID,
		Status:  "CREATED",
		Message: "Your request is being processed",
	})

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(resData)
}

func result(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if err := r.ParseForm(); err != nil {
		log.Error("Errror parsing form: ", err)
	}

	qs := r.URL.Query()

	// log.Debug("Check query:\n\n %v\n\n>> %d", r.URL.Query(), len(qs["job_id"][0]))

	if len(qs) == 0 {
		w.WriteHeader(400)
		w.Write([]byte(`{"status": "ERROR", "message": "Please provide a valid job id"}`))
		return
	}

	if qs["job_id"] == nil || len(qs["job_id"][0]) == 0 {
		w.WriteHeader(400)
		w.Write([]byte(`{"status": "ERROR", "message": "Please provide a valid job id"}`))
		return
	}

	// jobID, err := uuid.FromString(qs["job_id"])
	// if err != nil {
	// 	w.WriteHeader(400)
	// 	w.Write([]byte(`{"status": "ERROR", "message": "Please provide a valid job id"}`))
	// 	return
	// }

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error("failed to create client: ", err)
		return
	}
	defer client.Close()

	bkt := client.Bucket(DefaultBucket)

	jd, err := loadJobDetails(ctx, bkt, qs["job_id"][0])
	if err != nil {
		log.Error("Error loading job details: ", err)
		w.WriteHeader(400)
		w.Write([]byte(`{"status": "ERROR", "message": "Are you sure you have a valid Job ID and email? Please try again or contact us if the problem persists"}`))
		return
	}

	resData, _ := json.Marshal(&JobResult{
		JobID:   jd.JobID,
		Status:  "pruning",
		Message: "Please wait while we generate some samples",
	})
	if jd.Status == "COMPLETED" {
		resData, _ = json.Marshal(&JobResult{
			JobID:   jd.JobID,
			Status:  "completed",
			Message: "https://data.vertlife.org/pruned_treesets/" + jd.JobID + "/" + jd.JobID + ".zip",
		})
	} else if jd.Status == "ERROR" {
		resData, _ = json.Marshal(&JobResult{
			JobID:   jd.JobID,
			Status:  "completed",
			Message: "There was a problem generating samples. Please try again or contact us if the problem persists",
		})
	}

	w.Write(resData)
}

func worker(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	t, ok := r.Header["X-Appengine-Taskname"]
	if !ok || len(t[0]) == 0 {
		log.Println("Invalid Task: No X-Appengine-Taskname request header found")
		http.Error(w, "Bad Request - Invalid Task", http.StatusBadRequest)
		return
	}

	// jobID := r.FormValue("job_id")
	// Extract the request body for further task details.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("worker.ioutil.ReadAll: ", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
	jobID := string(body)

	fmt.Fprintf(w, "Prune worker job: %v\n\n", jobID)
	// DummyJob(ctx, jobID)
	_, err = StartPruneJob(ctx, jobID)
	if err != nil {
		log.Error("PRUNE ERROR: ", err)
		log.Info("Retrying job: ", jobID)

		jd, err2 := StartPruneJob(ctx, jobID)
		if err2 != nil {
			log.Error("PRUNE ERROR WITH RETRY: ", err2)
			if jd != nil {
				jd.SendJobFailedEmail(ctx)
			}
		}
	}
}
