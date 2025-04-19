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
curl -v --request DELETE "localhost:10086/tasks/ef36cdfa-be83-4116-98fe-8c7d3c2f2b73"

curl -v --request POST localhost:10086/tasks \
  --header 'Content-Type: application/json' \
  --data '
{
    "ID": "6be4cb6b-61d1-40cb-bc7b-9cacefefa60c",
    "State": 2,
    "Task": {
        "State": 1,
        "ID": "21b23589-5d2d-4731-b5c9-a97e9832d021",
        "Name": "test-chapter-5",
        "Image": "strm/helloworld-http"
    }
}'


curl -v --request POST localhost:10086/tasks \
  --header 'Content-Type: application/json' \
  --data '
{
    "ID": "6be4cb6b-61d1-40cb-bc7b-9cacefefa60c",
    "State": 3,
    "Task": {
        "State": 3,
        "ID": "21b23589-5d2d-4731-b5c9-a97e9832d021",
        "ContainerID": "b97526c353d5d9fc9092c32f1e4b5d6ad9c5fa88310c8d7e317663cf4d418e64",
        "Name": "test-chapter-5",
        "Image": "strm/helloworld-http"
    }
}
'
