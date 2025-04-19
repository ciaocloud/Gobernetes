package manager

import (
	"Gobernetes/task"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

type ErrResponse struct {
	Message    string //`json:"message"`
	HttpStatus int    // `json:"httpStatus"`
}

type ManagerAPI struct {
	Address string
	Port    int
	Manager *Manager
	Router  *chi.Mux
}

func (api *ManagerAPI) Start() {
	api.initRouter()
	http.ListenAndServe(fmt.Sprintf("%s:%d", api.Address, api.Port), api.Router)
}

func (api *ManagerAPI) initRouter() {
	api.Router = chi.NewRouter()
	api.Router.Route("/tasks", func(r chi.Router) {
		r.Get("/", api.GetTasksHandler)   // Get a list of all tasks
		r.Post("/", api.StartTaskHandler) // Create a new task
		r.Route("/{taskID}", func(r chi.Router) {
			r.Delete("/", api.StopTaskHandler) // Stop a task by taskID
		})
	})
}

func (api *ManagerAPI) StartTaskHandler(writer http.ResponseWriter, request *http.Request) {
	var taskEvent task.TaskEvent
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&taskEvent); err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf(msg)
		writer.WriteHeader(http.StatusBadRequest)
		errResp := ErrResponse{Message: msg, HttpStatus: http.StatusBadRequest}
		json.NewEncoder(writer).Encode(errResp)
		return
	}
	api.Manager.AddTask(taskEvent)
	writer.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(writer).Encode(taskEvent.Task); err != nil {
		log.Printf("Error encoding task: %v\n", err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
}

func (api *ManagerAPI) GetTasksHandler(writer http.ResponseWriter, request *http.Request) {
	tasks := api.Manager.GetTasks()
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(tasks); err != nil {
		log.Printf("Error encoding tasks: %v\n", err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
}

func (api *ManagerAPI) StopTaskHandler(writer http.ResponseWriter, request *http.Request) {
	taskID := chi.URLParam(request, "taskID")
	if taskID == "" {
		log.Printf("No task ID provided in the request.\n")
		writer.WriteHeader(http.StatusBadRequest)
	}
	taskUuid, _ := uuid.Parse(taskID)
	taskToStop, ok := api.Manager.TaskDb[taskUuid]
	if !ok {
		log.Printf("No task found with UUID %s\n", taskUuid)
		writer.WriteHeader(http.StatusNotFound)
	}
	taskCopy := *taskToStop
	taskCopy.State = task.Completed

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
		Task:      taskCopy,
	}
	api.Manager.AddTask(te)

	log.Printf("Added task event %v to stop task %v\n", te.ID, taskToStop.ID)
	writer.WriteHeader(http.StatusNoContent)
}
