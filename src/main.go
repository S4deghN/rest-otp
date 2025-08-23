package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"os"

	"rest-otp/db"
	"rest-otp/util"
)

const (
	OtpExpirationTime    = 2 * time.Minute
	OtpRequestTimeLimit  = 10 * time.Minute
	OtpRequestCountLimit = 3

	DefaultQueryPageLimit = 50
)

func populateOTP(o *db.Otp) (retryAfter time.Time) {
	if time.Now().Sub(o.FirstTry) > OtpRequestTimeLimit {
		o.FirstTry = time.Now()
		o.Tries = 0
	}

	if o.Tries < OtpRequestCountLimit {
		o.Tries += 1
	} else {
		retryAfter = o.FirstTry.Add(10 * time.Minute).UTC()
		return
	}

	if time.Now().Sub(o.ExpiresAt) > OtpExpirationTime {
		o.Val = util.GenerateOTP()
		o.ExpiresAt = time.Now().Add(OtpExpirationTime)
	}
	return
}

type Server struct {
	db       *db.DataBase
	syncChan chan struct{}
}

func NewServer(dbDriver, dbDsn string) (*Server, error) {
	dataBase, err := db.NewDb(dbDriver, dbDsn)
	return &Server{
		db:       dataBase,
		syncChan: make(chan struct{}, 1),
	}, err
}

func (s *Server) authOtpHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PhoneNumber string `json:"phone"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if data.PhoneNumber == "" { // TODO: Add phone number validity check.
		http.Error(w, "Missing or Invalid Phone Number", http.StatusBadRequest)
		return
	}

	id := data.PhoneNumber

	user, err := s.db.GetUser(id)
	if err != nil {
		log.Print(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	s.syncChan <- struct{}{}
	defer func() { <-s.syncChan }()

	if user == nil {
		user = &db.User{Id: id, State: db.UserStatePending}
		populateOTP(&user.Otp)
	} else {
		if user.State == db.UserStateLoggedIn {
			http.Error(w, "User Is Logged In", http.StatusBadRequest)
			return
		}

		retryAfter := populateOTP(&user.Otp)
		if !retryAfter.IsZero() {
			w.Header().Add("Retry-After", retryAfter.Format(http.TimeFormat))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
	}

	err = s.db.SaveUser(user)
	if err != nil {
		log.Print(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = util.SendOTP(user.Id, user.Otp.Val)
	if err != nil {
		http.Error(w, "Failed to Send OTP", http.StatusInternalServerError)
		log.Print(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

func (s *Server) authLoginHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PhoneNumber string `json:"phone"`
		Otp         string `json:"otp"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if data.PhoneNumber == "" || data.Otp == "" {
		http.Error(w, "Required JSON Fields Not Provided", http.StatusBadRequest)
		return
	}

	id := data.PhoneNumber
	otp := data.Otp

	user, err := s.db.GetUser(id)
	if err != nil {
		log.Print(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	s.syncChan <- struct{}{}
	defer func() { <-s.syncChan }()

	if user == nil {
		http.Error(w, "No Such Pending User", http.StatusBadRequest)
		return
	}

	if user.Otp.Val != otp {
		http.Error(w, "Invalid OTP", http.StatusBadRequest)
		return
	}

	if time.Now().Sub(user.Otp.ExpiresAt) > OtpExpirationTime {
		http.Error(w, "OTP Is Expired", http.StatusBadRequest)
		return
	}

	if user.RegDate.IsZero() {
		user.RegDate = time.Now()
	}
	user.State = db.UserStateLoggedIn

	err = s.db.SaveUser(user)
	if err != nil {
		log.Print(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	token, err := util.GenerateJWT(id)
	if err != nil {
		log.Print(err)
		http.Error(w, "Failed to Generate JWT", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func (s *Server) adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	limit, _ := strconv.ParseUint(r.FormValue("limit"), 0, 32)
	offset, _ := strconv.ParseUint(r.FormValue("offset"), 0, 32)

	w.Header().Set("Content-Type", "application/json")

	if id != "" {
		user, err := s.db.GetUser(id)
		if err != nil {
			log.Print(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(user)
		return
	}

	if limit == 0 {
		limit = DefaultQueryPageLimit
	}
	users, err := s.db.ListUsers(int(offset), int(limit))
	if err != nil {
		log.Print(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}

type httpHandle func(http.ResponseWriter, *http.Request)

func httpHandleWith(method, contentType string, f httpHandle) httpHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if contentType != "" && r.Header.Get("Content-Type") != contentType {
			http.Error(w, "Invalid Content Type", http.StatusBadRequest)
			return
		}

		f(w, r)
	}
}

func main() {
	serveAddr := ":8080"

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	server, err := NewServer("mysql", "root:db@tcp("+dbHost+":3306)/db?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/auth/otp-request", httpHandleWith("POST", "application/json", server.authOtpHandler))
	http.HandleFunc("/auth/login", httpHandleWith("POST", "application/json", server.authLoginHandler))
	http.HandleFunc("/admin/users", httpHandleWith("GET", "", server.adminUsersHandler))

	log.Printf("Serving on %s", serveAddr)
	err = http.ListenAndServe(serveAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
