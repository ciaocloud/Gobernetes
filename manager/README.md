In manager:
    - API: user interaction with the ochestration system
        - /api/v1/manager
        - /api/v1/manager/<id>
        - /api/v1/manager/<id>/status
    - Task Database: the manager maintains the states of all tasks in the system in a database.
    - Event Database: the manager maintains the states of all events (i.e., task.TaskEvent).
    - Task Queue: the manager maintains a queue of tasks that are waiting to be executed.
    - Workers: the manager keeps track of workers that are available to execute tasks.