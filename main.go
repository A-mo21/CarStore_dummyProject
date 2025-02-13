package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Car represents the structure of our car data
type Car struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Make  string             `bson:"make" json:"make"`
	Model string             `bson:"model" json:"model"`
	Year  int                `bson:"year" json:"year"`
	Price float64            `bson:"price" json:"price"`
}

// CarHandler handles all car-related operations
type CarHandler struct {
	collection *mongo.Collection
}

// Setup creates a new CarHandler with MongoDB connection
func Setup() (*CarHandler, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27018"))
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	collection := client.Database("carstore").Collection("cars")
	return &CarHandler{collection: collection}, nil
}

func createTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func (h *CarHandler) getAllCars(c *gin.Context) {
	ctx, cancel := createTimeout()
	defer cancel()

	var cars []Car
	cursor, err := h.collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cars"})
		return
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &cars)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load cars"})
		return
	}

	c.JSON(http.StatusOK, cars)
}

func (h *CarHandler) addCar(c *gin.Context) {
	ctx, cancel := createTimeout()
	defer cancel()

	var car Car
	if err := c.ShouldBindJSON(&car); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid car data"})
		return
	}

	result, err := h.collection.InsertOne(ctx, car)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add car"})
		return
	}

	car.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, car)
}

func (h *CarHandler) getCarByID(c *gin.Context) {
	ctx, cancel := createTimeout()
	defer cancel()

	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid car ID"})
		return
	}

	var car Car
	err = h.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&car)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Car not found"})
		return
	}

	c.JSON(http.StatusOK, car)
}

func (h *CarHandler) updateCar(c *gin.Context) {
	ctx, cancel := createTimeout()
	defer cancel()

	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid car ID"})
		return
	}

	var car Car
	if err := c.ShouldBindJSON(&car); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid car data"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"make":  car.Make,
			"model": car.Model,
			"year":  car.Year,
			"price": car.Price,
		},
	}
	result, err := h.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update car"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Car not found"})
		return
	}

	car.ID = id
	c.JSON(http.StatusOK, car)
}

func (h *CarHandler) deleteCar(c *gin.Context) {
	ctx, cancel := createTimeout()
	defer cancel()

	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid car ID"})
		return
	}

	result, err := h.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete car"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Car not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func main() {
	carHandler, err := Setup()
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	router.GET("/cars", carHandler.getAllCars)
	router.POST("/cars", carHandler.addCar)
	router.GET("/car/:id", carHandler.getCarByID)
	router.PUT("/car/:id", carHandler.updateCar)
	router.DELETE("/car/:id", carHandler.deleteCar)

	log.Println("Server starting on http://localhost:8080")
	router.Run(":8080")
}
