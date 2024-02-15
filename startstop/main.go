package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

func main() {
	// Load the location for CET (Central European Time)
	loc, err := time.LoadLocation("CET")
	if err != nil {
		panic(err)
	}

	// Available CET times
	cetTimes := []string{"7:00", "19:00", "08:26"}

	for {

		// Convert the UTC time to CET
		currentTimeInCET := time.Now().In(loc).Format("15:04")

		if Contains(cetTimes, currentTimeInCET) {
			RDS(currentTimeInCET)
			ECS(currentTimeInCET)
		}

		// Format and print the time in CET
		// fmt.Println("Current time in CET:", currentTimeInCET.Format("2006-01-02 15:04:05"))
		fmt.Println("Current time in CET:", currentTimeInCET)
		time.Sleep(60 * time.Second)
	}
}

// Resource Execution functions
func RDS(cetTime string) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("failed to load configuration : ", err)
		return
	}
	file, err := os.Open("rds.csv")
	if err != nil {
		log.Println("failed to open rds.csv file : ", err)
		return
	}
	defer file.Close()

	rdsClient := rds.NewFromConfig(cfg)

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Failed to read lines in rds.csv", err)
		return
	}
	for _, record := range records {
		fmt.Println(record)
		// record[0] == time, record[1] == action (start | stop), record[2] == name
		if record[0] == cetTime {
			if canPerformRds(rdsClient, record[2]) {
				if record[1] == "start" {
					RdsInstanceStart(rdsClient, record[2])
				} else if record[1] == "stop" {
					RdsInstanceStop(rdsClient, record[2])
				}
			}
		}
	}

}

func ECS(cetTime string) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("failed to load configuration : ", err)
		return
	}
	file, err := os.Open("ecs.csv")
	if err != nil {
		log.Println("failed to open rds.csv file : ", err)
		return
	}
	defer file.Close()

	ecsClient := ecs.NewFromConfig(cfg)

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Failed to read lines in rds.csv", err)
		return
	}

	for _, record := range records {
		fmt.Println(record)
		// record[0] == time, record[1] == action (start | stop), record[2] == clustername, record[3] == servicename
		if record[0] == cetTime {
			if canPerformEcs(ecsClient, record[2], record[3]) {
				if record[1] == "start" {
					EcsServiceStart(ecsClient, record[2], record[3], record[4])
				} else if record[1] == "stop" {
					EcsServiceStop(ecsClient, record[2], record[3])
				}
			}
		}
	}
}

// Utility Functions

func Contains(cetTimes []string, nowTime string) bool {
	for _, item := range cetTimes {
		if item == nowTime {
			return true
		}
	}
	return false
}

func canPerformRds(client *rds.Client, instanceIdentifier string) bool {
	arn := fmt.Sprintf("arn:aws:rds:%s:%s:db:%s", "eu-west-1", "", instanceIdentifier) // fill region, account id
	listTagsOutput, err := client.ListTagsForResource(context.TODO(), &rds.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})
	if err != nil {
		log.Printf("unable to list tags for resource, %v\n", err)
		return false
	}

	for _, tag := range listTagsOutput.TagList {
		// fmt.Printf("Key: %s, Value: %s\n", *tag.Key, *tag.Value)
		if *tag.Key == "startstop" && *tag.Value == "True" {
			return true
		}
	}

	return false
}

func canPerformEcs(client *ecs.Client, clustername string, servicename string) bool {
	service, err := client.DescribeServices(context.Background(), &ecs.DescribeServicesInput{
		Cluster:  aws.String(clustername),
		Services: []string{servicename},
	})
	if err != nil {
		log.Printf("unable to list ecs service, %v\n", err)
		return false
	}
	if len(service.Services) == 0 {
		fmt.Println("Service not found:", servicename)
		return false
	}

	if service.Services[0].Tags == nil {
		fmt.Println("Service has no tags.")
		return false
	}

	for _, tag := range service.Services[0].Tags {
		// fmt.Printf("Key: %s, Value: %s\n", *tag.Key, *tag.Value)
		if *tag.Key == "startstop" && *tag.Value == "True" {
			return true
		}
	}

	return false
}

func RdsInstanceStop(client *rds.Client, dbInstanceIdentifier string) {
	input := &rds.StopDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
	}

	result, err := client.StopDBInstance(context.Background(), input)
	if err != nil {
		log.Printf("Unable to stop RDS instance: %v\n", err)
		return
	}

	fmt.Printf("Successfully requested stop for instance: %v\n", *result.DBInstance.DBInstanceIdentifier)
}

func RdsInstanceStart(client *rds.Client, dbInstanceIdentifier string) {
	input := &rds.StartDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
	}

	result, err := client.StartDBInstance(context.Background(), input)
	if err != nil {
		log.Printf("Unable to start RDS instance: %v\n", err)
		return
	}

	fmt.Printf("Successfully requested start for instance: %v\n", *result.DBInstance.DBInstanceIdentifier)
}

func EcsServiceStart(client *ecs.Client, clustername string, servicename string, desiredcount string) {
	dc, err := strconv.Atoi(desiredcount)
	if err != nil {
		fmt.Println("string to int conversion failed")
		return
	}
	_, err = client.UpdateService(context.Background(), &ecs.UpdateServiceInput{
		Cluster:      aws.String(clustername),
		Service:      aws.String(servicename),
		DesiredCount: aws.Int32(int32(dc)),
	})
	if err != nil {
		fmt.Println("Not able to update ecs desired count", err)
		return
	}

	fmt.Println("Successfully updated service desired count set to : ", dc)
}

func EcsServiceStop(client *ecs.Client, clustername string, servicename string) {

	_, err := client.UpdateService(context.Background(), &ecs.UpdateServiceInput{
		Cluster:      aws.String(clustername),
		Service:      aws.String(servicename),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		fmt.Println("Not able to update ecs desired count", err)
		return
	}

	fmt.Println("Successfully updated service desired count set to : 0")
}
