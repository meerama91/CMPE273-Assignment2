package main

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Resource struct {
	Id         bson.ObjectId `bson:"_id" json:"id"`
	Name       string        `bson:"name" json:"name"`
	Address    string        `bson:"address" json:"address"`
	City       string        `bson:"city" json:"city"`
	State      string        `bson:"state" json:"state"`
	Zip        string        `bson:"zip" json:"zip"`
	Coordinate Coord         `bson:"coord" json:"coord"`
}

type Coord struct {
	Lat  float64
	Long float64
}

//var session *mgo.Session
type UserController struct {
	session *mgo.Session
}

type googleGeocodeResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64
				Lng float64
			}
		}
	}
}

var uc UserController
var uri string

func getSession() *mgo.Session {
	// Connect to our local mongo
	s, err := mgo.Dial(uri)

	// Check if connection error, is mongo running?
	if err != nil {
		panic(err)
	}
	return s
}
func NewUserController(s *mgo.Session) UserController {
	return UserController{s}
}

func getCoord(url string) Coord {
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	res2 := &googleGeocodeResponse{}
	json.Unmarshal(body, res2)
	if len(res2.Results) == 0 {
		log.Fatal(err)
	}
	lat := res2.Results[0].Geometry.Location.Lat
	lng := res2.Results[0].Geometry.Location.Lng
	result := Coord{
		Lat:  lat,
		Long: lng,
	}
	return result
}

//func locations(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
//  fmt.Fprintf(rw, "Hello, %s!\n", p.ByName("name"))
//}

func getter(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	id := p.ByName("location_id")
	if !bson.IsObjectIdHex(id) {
		rw.WriteHeader(404)
		return
	}
	fmt.Println("id id ", id)
	oid := bson.ObjectIdHex(id)
	fmt.Println("oid id ", oid)
	response := Resource{}
	//  c:=uc.session.DB("locations").C("locations")
	err := uc.session.DB("locations").C("locations").FindId(oid).One(&response)

	if err != nil {
		rw.WriteHeader(404)
		return
	}
	responseJson, _ := json.Marshal(response)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	fmt.Fprintf(rw, "%s", responseJson)
}

func creator(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	request := Resource{}
	response := Resource{}
	json.NewDecoder(req.Body).Decode(&request)

	response.Name = request.Name
	response.Address = request.Address
	response.City = request.City
	response.State = request.State
	response.Zip = request.Zip
	response.Id = bson.NewObjectId()
	urladd := strings.Replace(request.Address, " ", "+", -1)
	urlcity := strings.Replace(request.City, " ", "+", -1)
	url1 := "http://maps.google.com/maps/api/geocode/json?address="
	url2 := "&sensor=false"
	url := url1 + urladd + ",+" + urlcity + ",+" + request.State + url2
	response.Coordinate = getCoord(url)

	//1600+Amphitheatre+Parkway,+Mountain+View,+CA&sensor=false
	err := uc.session.DB("locations").C("locations").Insert(response)
	//c := session.DB("locations").C("locations")
	//  err := c.Insert(response)
	if err != nil {
		rw.WriteHeader(404)
		return
	}
	responseJson, _ := json.Marshal(response)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(201)
	fmt.Fprintf(rw, "%s", responseJson)
}

func updater(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	id := p.ByName("location_id")
	request := Resource{}
	
	json.NewDecoder(req.Body).Decode(&request)
	
	if !bson.IsObjectIdHex(id) {
		rw.WriteHeader(404)
		return
	}
	
	response := Resource{}
	oid := bson.ObjectIdHex(id)
	Query:= bson.M{"_id": oid}
	urladd := strings.Replace(request.Address, " ", "+", -1)
	urlcity := strings.Replace(request.City, " ", "+", -1)
	url1 := "http://maps.google.com/maps/api/geocode/json?address="
	url2 := "&sensor=false"
	url := url1 + urladd + ",+" + urlcity + ",+" + request.State + url2
	newCord := getCoord(url)
	change:=bson.M{"$set": bson.M{"address": request.Address, "city": request.City, "state" : request.State, "zip" : request.Zip, "coord":newCord}}
	
	err := uc.session.DB("locations").C("locations").Update(Query, change)
	

	if err != nil {
		rw.WriteHeader(404)
		return
	}
	
	err = uc.session.DB("locations").C("locations").FindId(oid).One(&response)
	if err != nil {
		rw.WriteHeader(404)
		return
	}
	responseJson, _ := json.Marshal(response)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	fmt.Fprintf(rw, "%s", responseJson)

}

func deletter(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	id := p.ByName("location_id")
	if !bson.IsObjectIdHex(id) {
		rw.WriteHeader(404)
		return
	}
	fmt.Println("id id ", id)
	oid := bson.ObjectIdHex(id)
	fmt.Println("oid id ", oid)
	// response := Resource{}
	//  Remove user.
	if err := uc.session.DB("locations").C("locations").RemoveId(oid); err != nil {
		rw.WriteHeader(404)
		return
	}

	// Write status
	rw.WriteHeader(200)
}

func main() {
	mux := httprouter.New()
	uri = "mongodb://muser:mpassword@ds029804.mongolab.com:29804/locations"
	uc = NewUserController(getSession())
	
	mux.POST("/locations", creator)
	mux.GET("/locations/:location_id", getter)
	mux.DELETE("/locations/:location_id", deletter)
	mux.PUT("/locations/:location_id", updater)
	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
