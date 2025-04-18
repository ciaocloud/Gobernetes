curl -v --request POST localhost:10086/tasks \
  --header 'Content-Type: application/json' \
  --data '
  {
    "ID": "266592cd-960d-4091-981c-8c25c44b1018",
    "State": 2,
    "Task": {
      "State": 1,
      "ID": "266592cd-960d-4091-981c-8c25c44b1018",
      "Name": "test-task",
      "Image": "strm/helloworld-http"
    }
  }'


curl -v --request DELETE "localhost:10086/tasks/266592cd-960d-4091-981c-8c25c44b1018"