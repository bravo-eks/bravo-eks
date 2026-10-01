module github.com/CoderCo-Learning/ecs-v3/services/shipping-service

go 1.22

require github.com/lib/pq v1.10.9

go get github.com/aws/aws-sdk-go-v2/config@v1.29.7
go get github.com/aws/aws-sdk-go-v2/service/sqs@v1.38.1
go mod tidy