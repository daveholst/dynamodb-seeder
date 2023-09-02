package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// Recursive attribuite builder for getting the JSON into the correct schema
// with attributes for seeding into dynamoDB
func buildAttributeValues(value interface{}) *dynamodb.AttributeValue {
	attributeValue := &dynamodb.AttributeValue{}

	switch v := value.(type) {
	case string:
		attributeValue.S = aws.String(v)
	case bool:
		attributeValue.BOOL = aws.Bool(v)
	case float64:
		attributeValue.N = aws.String(fmt.Sprintf("%f", v))
	case int:
		attributeValue.N = aws.String(fmt.Sprintf("%d", v))
	case []interface{}:
		l := make([]*dynamodb.AttributeValue, len(v))
		for i, item := range v {
			l[i] = buildAttributeValues(item)
		}
		attributeValue.L = l
	case map[string]interface{}:
		m := make(map[string]*dynamodb.AttributeValue)
		for k, item := range v {
			m[k] = buildAttributeValues(item)
		}
		attributeValue.M = m
	default:
		attributeValue.NULL = aws.Bool(true)
	}

	return attributeValue
}

func main() {
	// TODO take this in as an arg
	filePath := "./fixtures/tours.json"
	// Flags
	host := flag.String("h", "http://localhost:8000", "DyanamoDB host to target")
	tableName := flag.String("t", "TestSingleTable", "Table name. Will create if doesn't exist")
	flag.Parse()

	// Create a new DynamoDB session
	session, err := session.NewSession(&aws.Config{
		Region:   aws.String("ap-southeast-2"), // Replace with your desired region
		Endpoint: aws.String(*host),            // Replace with the URL of your DynamoDB Local instance
	})

	if err != nil {
		fmt.Printf("Failed to create session: %v\n", err)
		return
	}

	// Create DynamoDB client
	dbClient := dynamodb.New(session)

	// TODO Allow a JSON table schema to be passed and then use this as default?
	// This schema matches the singleTable constructor we made for CDK
	// Specify the table name and its attributes
	attributeDefinitions := []*dynamodb.AttributeDefinition{
		{
			AttributeName: aws.String("pk"),
			AttributeType: aws.String("S"),
		},
		{
			AttributeName: aws.String("sk"),
			AttributeType: aws.String("S"),
		},
		{
			AttributeName: aws.String("GSI1PK"),
			AttributeType: aws.String("S"),
		},
		{
			AttributeName: aws.String("GSI1SK"),
			AttributeType: aws.String("S"),
		},
		{
			AttributeName: aws.String("GSI2PK"),
			AttributeType: aws.String("S"),
		},
		{
			AttributeName: aws.String("GSI2SK"),
			AttributeType: aws.String("S"),
		},
	}
	keySchema := []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String("pk"),
			KeyType:       aws.String("HASH"),
		},
		{
			AttributeName: aws.String("sk"),
			KeyType:       aws.String("RANGE"),
		},
	}
	provisionedThroughput := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(1),
		WriteCapacityUnits: aws.Int64(1),
	}
	gsi1 := &dynamodb.GlobalSecondaryIndex{
		IndexName: aws.String("GSI1"),
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("GSI1PK"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("GSI1SK"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: provisionedThroughput,
		Projection: &dynamodb.Projection{
			ProjectionType: aws.String("ALL"),
		},
	}
	gsi2 := &dynamodb.GlobalSecondaryIndex{
		IndexName: aws.String("GSI2"),
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("GSI2PK"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("GSI2SK"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: provisionedThroughput,
		Projection: &dynamodb.Projection{
			ProjectionType: aws.String("ALL"),
		},
	}

	// Check if the table already exists
	// Call the DescribeTable API to check if the table exists
	_, err = dbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(*tableName),
	})

	if err != nil {
		// Check if the error is a ResourceNotFoundException
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
			fmt.Printf("Table with name '%s' does not exist. Creating...\n", *tableName)

			// Create the createInput for the CreateTable API call
			createInput := &dynamodb.CreateTableInput{
				TableName:             aws.String(*tableName),
				AttributeDefinitions:  attributeDefinitions,
				KeySchema:             keySchema,
				ProvisionedThroughput: provisionedThroughput,
				GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
					gsi1,
					gsi2,
				},
			}

			// Create the table in DynamoDB
			_, err = dbClient.CreateTable(createInput)
			if err != nil {
				fmt.Printf("Failed to create table %v\n", err)
				return
			} else {
				fmt.Println("Table creation request submitted!")
			}

			// Wait for the table to be created
			err = dbClient.WaitUntilTableExists(&dynamodb.DescribeTableInput{
				TableName: aws.String(*tableName),
			})
			if err != nil {
				fmt.Printf("Failed to wait for table creation: %v\n", err)
				return
			} else {
				fmt.Println("Table created successfully!")
			}
		} else {
			fmt.Printf("Error describing table %v\n", err)
		}
	} else {
		fmt.Printf("Table with name '%s' found.\n", *tableName)

	}

	// Read the JSON file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var itemMap []map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &itemMap)
	if err != nil {
		panic(err)
	}

	// Create the input for the PutItem API call
	item := make(map[string]*dynamodb.AttributeValue)
	for _, itemData := range itemMap {
		for key, value := range itemData {
			attributeValue := buildAttributeValues(value)
			item[key] = attributeValue
		}
	}

	// TODO I think this will do a blind rewrite, might need a flag to control this
	putInput := &dynamodb.PutItemInput{
		TableName: aws.String(*tableName), // Replace with your table name
		Item:      item,
	}

	// Call the PutItem API to add the item to the table
	_, err = dbClient.PutItem(putInput)
	if err != nil {
		fmt.Printf("Failed to put item: %v\n", err)
		return
	}

	fmt.Println("Item added successfully!")

}
