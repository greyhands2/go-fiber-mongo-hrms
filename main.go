package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const connectionString string = "mongodb://localhost:27017"
const colName = "employees"

//most important

var collection *mongo.Collection

const dbName = "fiber-hrms"

type Employee struct {
	ID     primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name   string             `json:"name"`
	Salary float64            `json:"salary"`
	Age    int32              `json:"age"`
}

func connect() error {
	//client option
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(connectionString))
	if err != nil {
		panic(err)
	}

	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}

	collection = client.Database(dbName).Collection(colName)
	return nil
}

func main() {

	//we can omit the _ here because the connect function we defined has a return type that's only error
	if err := connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(reqRes *fiber.Ctx) error {
		//since we are feteching all employees our query is empty
		query := bson.M{}

		cursor, err := collection.Find(reqRes.Context(), query)

		if err != nil {
			return reqRes.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0)

		//convert cursor data from mongodb  to slice data for golang to understand
		if err := cursor.All(reqRes.Context(), &employees); err != nil {
			return reqRes.Status(500).SendString(err.Error())
		}

		defer cursor.Close(reqRes.Context())
		return reqRes.JSON(employees)

	})
	app.Post("/employee", func(reqRes *fiber.Ctx) error {

		//another way of initializing an Employee instance
		employee := new(Employee)

		if err := reqRes.BodyParser(employee); err != nil {
			return reqRes.Status(400).SendString(err.Error())
		}

		insertionRes, err := collection.InsertOne(reqRes.Context(), employee)

		if err != nil {
			return reqRes.Status(500).SendString(err.Error())
		}

		filter := bson.M{"_id": insertionRes.InsertedID}

		createdRecord := collection.FindOne(reqRes.Context(), filter)
		//we have to decode it into a new employee struct instance
		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return reqRes.Status(200).JSON(createdEmployee)

	})
	app.Put("/employee/:id", func(reqRes *fiber.Ctx) error {
		idParam := reqRes.Params("id")
		empID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return reqRes.SendStatus(400)
		}

		var employee Employee
		if err := reqRes.BodyParser(&employee); err != nil {
			return reqRes.Status(400).SendString(err.Error())
		}

		//update query
		query := bson.M{"_id": empID}
		update := bson.M{

			"$set": bson.M{
				"name":   employee.Name,
				"age":    employee.Age,
				"salary": employee.Salary,
			},
		}

		err = collection.FindOneAndUpdate(reqRes.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return reqRes.SendStatus(400)
			}
			return reqRes.SendStatus(500)
		}

		return reqRes.Status(200).JSON(employee)

	})
	app.Get("/employee/:id", func(reqRes *fiber.Ctx) error {
		empId, err := primitive.ObjectIDFromHex(reqRes.Params("id"))

		if err != nil {
			return reqRes.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: empId}}

		var emp Employee
		err = collection.FindOne(reqRes.Context(), query).Decode(&emp)

		if err != nil {
			return reqRes.SendStatus(404)
		}

		return reqRes.Status(200).JSON(emp)
	})

	app.Delete("/employee/:id", func(reqRes *fiber.Ctx) error {

		empId, err := primitive.ObjectIDFromHex(reqRes.Params("id"))

		if err != nil {
			return reqRes.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: empId}}

		res, err := collection.DeleteOne(reqRes.Context(), query)

		if err != nil {
			return reqRes.SendStatus(500)
		}

		if res.DeletedCount < 1 {
			return reqRes.SendStatus(404)
		}

		return reqRes.Status(200).JSON("Sucessfully deleted")

	})

	log.Fatal(app.Listen(":3000"))
}
