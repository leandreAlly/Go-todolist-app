package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	rnd    *renderer.Render
	client *mongo.Client
	db     *mongo.Database
)

const (
	dbName         string = "golang-todo"
	collectionName string = "todo"
)

type (
	TodoModel struct {
		ID        primitive.ObjectID `bson:"id,omitempty"`
		Title     string             `bson:"title"`
		Completed bool               `bson:"completed"`
		CreatedAt time.Time          `bson:"created_at"`
	}

	Todo struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed bool      `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}

	GetTodoResponse struct {
		Message string `json:"message"`
		Data    []Todo `json:"data"`
	}

	CreateTodo struct {
		Title string `json:"title"`
	}

	UpdateTodo struct {
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}
)

func init() {
	fmt.Println("init function running")

	rnd = renderer.New()
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	checkError(err)

	err = client.Ping(ctx, readpref.Primary())
	checkError(err)

	db = client.Database(dbName)
}

func homeHandler(rw http.ResponseWriter, r *http.Request) {
	filepath := "./README.md"
	err := rnd.FileView(rw, http.StatusOK, filepath, "readme.md")
	checkError(err)
}

func getTodos(rw http.ResponseWriter, r *http.Request) {
	todoListFromDB := []TodoModel{}
	filter := bson.D{}

	cursor, err := db.Collection(collectionName).Find(context.Background(), filter)
	if err != nil {
		log.Printf("failed to fetch todo records from the db: %v\n", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "Could not fetch the todo collection",
			"error":   err.Error(),
		})
	}

	todoList := []Todo{}
	if err = cursor.All(context.Background(), &todoListFromDB); err != nil {
		checkError(err)
	}

	// loop through the database list, convert TodoModel to JSON and append to the todoList array.
	for _, td := range todoListFromDB {
		todoList = append(todoList, Todo{
			ID:        td.ID.Hex(),
			Title:     td.Title,
			Completed: td.Completed,
			CreatedAt: td.CreatedAt,
		})
	}

	rnd.JSON(rw, http.StatusOK, GetTodoResponse{
		Message: "All todos retrived",
		Data:    todoList,
	})

	return
}

func createTodo(rw http.ResponseWriter, r *http.Request) {
	var todoReq CreateTodo

	if err := json.NewDecoder(r.Body).Decode(&todoReq); err != nil {
		log.Printf("failed to decode json data: %v\n", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "couldnot decode data",
		})

		return
	}

	if todoReq.Title == "" {
		log.Println("not title added to response body")
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "please add a title",
		})
		return
	}

	TodoModel := TodoModel{
		ID:        primitive.NewObjectID(),
		Title:     todoReq.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}

	// add the todo to the database
	data, err := db.Collection(collectionName).InsertOne(r.Context(), TodoModel)
	if err != nil {
		log.Printf("failed to insert data into the database: %v\n", err.Error())
		rnd.JSON(rw, http.StatusInternalServerError, renderer.M{
			"message": "Failed to insert data into the database",
			"error":   err.Error(),
		})
		return
	}

	rnd.JSON(rw, http.StatusCreated, renderer.M{
		"messsage": "Todo created successfully",
		"ID":       data.InsertedID,
	})
}

func updateTodo(rw http.ResponseWriter, r *http.Request) {
	// get the id from the url params
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	res, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("The id param is not valid hex value: %v\n", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "The id is invalid",
			"error":   err.Error(),
		})
		return
	}

	var updateTodoReq UpdateTodo

	if err := json.NewDecoder(r.Body).Decode(&updateTodoReq); err != nil {
		log.Print("failed to decode the json response body data: %v\n", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, err.Error())
	}

	if updateTodoReq.Title == "" {
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "Title cannot be empty",
		})

		return
	}

	// update the todo in the database
	filter := bson.M{"id": res}
	update := bson.M{"$set": bson.M{"title": updateTodoReq.Title, "completed": updateTodoReq.Completed}}
	data, err := db.Collection(collectionName).UpdateOne(r.Context(), filter, update)
	if err != nil {
		log.Printf("failed to update db collection: %v\n", err.Error())
		rnd.JSON(rw, http.StatusInternalServerError, renderer.M{
			"message": "Failed to update data in the database",
			"error":   err.Error(),
		})

		return
	}

	rnd.JSON(rw, http.StatusOK, renderer.M{
		"message": "Todo update successfully",
		"data":    data.ModifiedCount,
	})
}

func deleteTodo(rw http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("Invalid id: %v\n", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, err.Error())

		return
	}

	filter := bson.M{"id": res}
	if data, err := db.Collection(collectionName).DeleteOne(r.Context(), filter); err != nil {
		log.Printf("could not delete item from database: %v\n", err.Error())
		rnd.JSON(rw, http.StatusInternalServerError, renderer.M{
			"message": "an error occured while deleting todo item",
			"error":   err.Error(),
		})
	} else {
		rnd.JSON(rw, http.StatusOK, renderer.M{
			"message": "item deleted successfuly ",
			"data":    data,
		})
	}
}

func main() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/", homeHandler)
	router.Mount("/todo", todoHandler())

	server := &http.Server{
		Addr:         ":9000",
		Handler:      chi.NewRouter(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// create a channel to receive signal
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	// start the server in a separate go routine
	go func() {
		fmt.Println("Server started on port", 9000)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("listen:%s\n", err)
		}
	}()

	sig := <-stopChan
	log.Printf("signal received: %v\n", sig)

	// disconnect mongo client from the database
	if err := client.Disconnect(context.Background()); err != nil {
		panic(err)
	}

	// create a context witha timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// shutdown the server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v\n", err)
	}
	log.Println("Server shutdown gracefully")

	// start the server
	fmt.Println("Server started on port", 9000)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("listen: %s\n", err)
	}
}

func todoHandler() http.Handler {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Get("/", getTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return router
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
