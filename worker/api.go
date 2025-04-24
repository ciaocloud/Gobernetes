package worker

import (
	"Gobernetes/task"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"log"
	"net/http"
)

/**
 * Worker API receives http requests from the manager, handles accordingly, and responds.
 * Exposed API endpoints:
 * 1. HTTP GET /tasks
 * 2. HTTP POST /tasks
 * 3. HTTP DELETE /tasks/{taskId}
 */

type ErrResponse struct {
	Message    string //`json:"message"`
	HttpStatus int    // `json:"httpStatus"`
}

type WorkerAPI struct {
	// Worker <-> Handler <-> Router <-> Routes <-> HTTP Server <-> Client (e.g. Manager)
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}

func (api *WorkerAPI) Start() {
	api.Router = chi.NewRouter()
	api.Router.Route("/tasks", func(r chi.Router) {
		r.Get("/", api.GetTasksHandler)   // Get a list of all tasks
		r.Post("/", api.StartTaskHandler) // Create a new task
		r.Route("/{taskId}", func(r chi.Router) {
			r.Delete("/", api.StopTaskHandler) // Stop a task by taskID
		})
	})
	api.Router.Route("/stats", func(r chi.Router) {
		r.Get("/", api.GetStatsHandler) // Get the stats of the worker
	})

	// Start the HTTP server
	//go func() {
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", api.Address, api.Port), api.Router)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	//}()
}

func (api *WorkerAPI) StartTaskHandler(w http.ResponseWriter, req *http.Request) {
	// Parse the request body
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	taskEvent := &task.TaskEvent{}
	err := decoder.Decode(taskEvent)
	if err != nil {
		msg := fmt.Sprintf("Failed unmarshalling body to a task event: %v\n", err)
		w.WriteHeader(http.StatusBadRequest) // code 400
		errResp := ErrResponse{Message: msg, HttpStatus: http.StatusBadRequest}
		json.NewEncoder(w).Encode(errResp)
		return
	}
	newTask := taskEvent.Task
	api.Worker.AddTask(&newTask)
	log.Printf("New task added: %v", newTask.ID)
	w.WriteHeader(http.StatusCreated)
	encoder := json.NewEncoder(w)
	if err = encoder.Encode(newTask); err != nil {
		log.Printf("Failed to encode task: %v", err)
		w.WriteHeader(http.StatusInternalServerError) // code 500
	}
}

func (api *WorkerAPI) InspectTaskHandler(writer http.ResponseWriter, request *http.Request) {
	taskId := chi.URLParam(request, "taskID")
	if taskId == "" {
		log.Printf("Missing task ID in request. \n")
		writer.WriteHeader(http.StatusBadRequest) // code 400
		return
	}
	taskUuid, _ := uuid.Parse(taskId)
	val, err := api.Worker.Db.Get(taskUuid.String())
	if err != nil {
		log.Printf("Task %s not found in the database. \n", taskId)
		writer.WriteHeader(http.StatusNotFound) // code 404
		return
	}
	taskToInspect := val.(*task.Task)
	resp := api.Worker.InspectTask(taskToInspect)
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // code 200
	encoder := json.NewEncoder(writer)
	err = encoder.Encode(resp)
	if err != nil {
		log.Printf("Failed to encode task: %v", err)
		writer.WriteHeader(http.StatusInternalServerError) // code 500
	}
}

func (api *WorkerAPI) GetTasksHandler(writer http.ResponseWriter, _ *http.Request) {
	tasks := api.Worker.GetTasks()
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // code 200
	encoder := json.NewEncoder(writer)
	err := encoder.Encode(tasks)
	if err != nil {
		log.Printf("Failed to encode tasks: %v", err)
		writer.WriteHeader(http.StatusInternalServerError) // code 500
	}
}

func (api *WorkerAPI) StopTaskHandler(writer http.ResponseWriter, request *http.Request) {
	taskId := chi.URLParam(request, "taskID")
	if taskId == "" {
		log.Printf("Missing task ID in request. \n")
		writer.WriteHeader(http.StatusBadRequest) // code 400
		return
	}
	taskUuid, _ := uuid.Parse(taskId)
	//taskToStop, ok := api.Worker.Db[taskUuid]
	val, err := api.Worker.Db.Get(taskUuid.String())
	if err != nil {
		log.Printf("Task %s not found in the database. \n", taskId)
		writer.WriteHeader(http.StatusNotFound) // code 404
		return
	}
	taskToStop := val.(*task.Task)
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	api.Worker.AddTask(&taskCopy)

	log.Printf("Added task %v to stop the container %v.\n", taskToStop.ID, taskToStop.ContainerID)
	writer.WriteHeader(http.StatusNoContent) // code 204
}

func (api *WorkerAPI) GetStatsHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // code 200
	encoder := json.NewEncoder(writer)
	err := encoder.Encode(api.Worker.Stats)
	if err != nil {
		log.Printf("Failed to encode stats: %v", err)
		writer.WriteHeader(http.StatusInternalServerError) // code 500
	}
}
