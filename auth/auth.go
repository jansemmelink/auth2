package auth

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bitbucket.org/conorit/golib-logger"
	types "bitbucket.org/conorit/golib-types"
	"github.com/gorilla/pat"
	"gopkg.in/mgo.v2/bson"
)

var (
	log                    = logger.New("auth")
	dbUserCollection       = Db().DB("auth").C("users")
	errUserAlreadyExists   = log.Errorf(nil, "User already exists")
	errUserDoesNotExist    = log.Errorf(nil, "User does not exist")
	errTempPasswordExpired = log.Errorf(nil, "Temp password expired.")
	errWrongPassword       = log.Errorf(nil, "Wrong password")
)

//AddAuthRoutes add the auth API to the router
func AddAuthRoutes(r *pat.Router) {
	//auth operations
	r.Post("/auth/register", registerHandler)

	r.Get("/auth/reset", resetHandler)
	r.Post("/auth/reset", resetHandler)

	r.Get("/auth/activate", activateHandler)
	r.Post("/auth/activate", activateHandler)

	r.Get("/auth/login", loginHandler)
	r.Post("/auth/login", loginHandler)

	r.Get("/auth/logout", logoutHandler)
	r.Post("/auth/logout", logoutHandler)
}

//User is what we store for an authentication entry
//TempPassword is not encrypted as it does not reveal anything about the user - its a random string
//TempPassword expires in a few minutes
//RealPassword is encrypted as SHA-1 when stored in the database
type User struct {
	ID           bson.ObjectId `bson:"_id" json:"_id"`
	Name         string
	Password     string
	TempPassword string
	TempExpiry   time.Time
}

//Insert to insert into the database - will check duplicate key
func (u User) Insert() (User, error) {
	//user.Name must be unique
	if _, err := u.getByName(u.Name); err == nil {
		return u, errUserAlreadyExists
	}

	//assign new ID
	u.ID = bson.NewObjectId()
	log.Debug.Printf("Creating user.id=%v", u.ID)

	//insert into the database
	if err := dbUserCollection.Insert(u); err != nil {
		return u, log.Errorf(err, "Failed on db.insert(%+v)", u)
	}

	log.Info.Printf("Created(%+v)", u)
	return u, nil
}

//Get to retrieve from the database by hex string ID
func (u User) Get(id string) (User, error) {
	log.Debug.Printf("Getting user.id=%s", id)
	if !bson.IsObjectIdHex(id) {
		return User{}, log.Errorf(nil, "Invalid id='%s' is not bson hex object id", id)
	}
	mgoKey := make(bson.M)
	mgoKey["_id"] = bson.ObjectIdHex(id)
	if err := dbUserCollection.Find(mgoKey).One(u); err != nil {
		return User{}, log.Errorf(err, "Failed to get")
	}
	return u, nil
} //User.Get()

func (u User) getByName(name string) (User, error) {
	log.Debug.Printf("Getting user.name=%s", name)
	mgoKey := make(bson.M)
	mgoKey["name"] = name
	if err := dbUserCollection.Find(mgoKey).One(&u); err != nil {
		return User{}, log.Errorf(err, "User(name=%s) does not exist", name)
	}
	return u, nil
} //user.getByName()

//Authenticate checks the Name + TempPassword/Password as specified against the database
//and on success, sets the ID and clears the password fields of output
func (u User) Authenticate() (User, error) {
	//load user by name
	existingUser := User{}
	var err error
	if u.ID.Valid() {
		existingUser, err = u.Get(u.ID.Hex())
	} else {
		existingUser, err = u.getByName(u.Name)
	}
	if err != nil {
		return u, errUserDoesNotExist
	}

	//check specified password, temp/real
	if u.TempPassword != "" {
		if existingUser.TempExpiry.Before(time.Now()) {
			return u, errTempPasswordExpired
		}
		if u.TempPassword != existingUser.TempPassword {
			return u, errWrongPassword
		}
		//authenticated inactive user
		//clear tempPassword so that subsequent
		//update will reset it and can define the actual password
		u.TempExpiry = time.Now()
		u.TempPassword = ""
	} else {
		//specified password is clear but stored password is encrypted
		//so write specified as encrypted then compare with stored
		pwHash := sha1.New()
		io.WriteString(pwHash, u.Password)
		pwSha1 := fmt.Sprintf("%x", pwHash.Sum(nil))
		if pwSha1 != existingUser.Password {
			return u, errWrongPassword
		}

		//authenticated active user, copy encrypted password
		//so that updates on user won't reset the password
		u.Password = existingUser.Password
	}

	//authenticated: include ID in output
	//do not include password value loaded from the database
	u.ID = existingUser.ID
	return u, nil
} //User.Authenticate()

//Update the user record in the database
//ID field must already be set, from Authenticate() or Insert()
func (u User) update() (User, error) {
	log.Debug.Printf("Updating user=%+v", u)

	err := dbUserCollection.UpdateId(u.ID, u)
	if err != nil {
		return u, log.Errorf(err, "Failed to db.update user %+v", u)
	}
	log.Info.Printf("Updated user %+v", u)
	return u, nil
} //User.Update()

func registerHandler(res http.ResponseWriter, req *http.Request) {
	jsonDecoder := json.NewDecoder(req.Body)
	user := User{}
	if err := jsonDecoder.Decode(&user); err != nil {
		http.Error(res, fmt.Sprintf("Invalid JSON: %v", err.Error()), http.StatusBadRequest)
		return
	}
	log.Debug.Printf("Register: user.Name=%s", user.Name)

	//validate reqData for this operation
	if user.Name == "" {
		http.Error(res, fmt.Sprintf("Invalid Request: missing UserName"), http.StatusBadRequest)
		return
	}

	//prepare new userData with random temp password for activation
	//and no real password
	user.Password = ""
	user.TempPassword = types.GeneratePassword(types.PasswordSpecification{Length: 8, Hex: true})
	user.TempExpiry = time.Now().Add(time.Hour * 1)

	//create the user in the database
	user, err := user.Insert()
	if err != nil {
		if err == errUserAlreadyExists {
			http.Error(res, fmt.Sprintf("User already exists"), http.StatusConflict)
		} else {
			http.Error(res, fmt.Sprintf("Failed to register: %v", err.Error()), http.StatusBadRequest)
		}
		return
	}
	log.Info.Printf("Registered user.Name=%s with ID=%s", user.Name, user.ID.Hex())

	jsonData, err := json.Marshal(user)
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to encode response: %v", err.Error()), http.StatusBadRequest)
		return
	}
	res.Write(jsonData)
} //registerHandler()

func resetHandler(res http.ResponseWriter, req *http.Request) {
	//request data either POSTed or in GET URL
	user := User{}
	if req.Method == http.MethodGet {
		user.Name = req.URL.Query().Get("name")
	} else {
		jsonDecoder := json.NewDecoder(req.Body)
		if err := jsonDecoder.Decode(&user); err != nil {
			http.Error(res, fmt.Sprintf("Invalid JSON: %v", err.Error()), http.StatusBadRequest)
			return
		}

		//reset unexpected fields
		user.TempExpiry = time.Now()
		user.TempPassword = ""
		user.Password = ""
	}
	log.Debug.Printf("reset: name=%s", user.Name)

	//load existing user by name
	var err error
	user, err = user.getByName(user.Name)
	if err != nil {
		http.Error(res, fmt.Sprintf("%v", err.Error()), http.StatusNotFound)
		return
	}

	log.Debug.Printf("Reset password for existing user %s", user.Name)

	//create new temp password, but leave existing password as is
	//in case user remembers it
	user.TempPassword = types.GeneratePassword(types.PasswordSpecification{Length: 8, Hex: true})
	user.TempExpiry = time.Now().Add(time.Hour * 1)

	//update the user in the database
	user, err = user.update()
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to reset password: %v", err.Error()), http.StatusBadRequest)
		return
	}
	log.Info.Printf("Reset done user.Name=%s with ID=%s", user.Name, user.ID.Hex())
	jsonData, err := json.Marshal(user)
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to encode response: %v", err.Error()), http.StatusBadRequest)
		return
	}
	res.Write(jsonData)
} //resetHandler()

func activateHandler(res http.ResponseWriter, req *http.Request) {
	//request data either POSTed or in GET URL
	user := User{}
	newPassword := ""
	if req.Method == http.MethodGet {
		user.Name = req.URL.Query().Get("name")
		user.TempPassword = req.URL.Query().Get("tpw")
		newPassword = req.URL.Query().Get("password")
	} else {
		jsonDecoder := json.NewDecoder(req.Body)
		if err := jsonDecoder.Decode(&user); err != nil {
			http.Error(res, fmt.Sprintf("Invalid JSON: %v", err.Error()), http.StatusBadRequest)
			return
		}

		//move new password out of user data
		newPassword = user.Password
		user.Password = ""
	}
	log.Debug.Printf("activate: name=%s tpw=%s npw=%s", user.Name, user.TempPassword, newPassword)

	//make sure the new password will be strong enough
	_, err := types.CheckPassword(newPassword, &types.PasswordSpecification{Length: 8, Lower: true, Upper: true, Digit: true})
	if err != nil {
		http.Error(res, fmt.Sprintf("New password is not strong enough: %v", err.Error()), http.StatusBadRequest)
		return
	}

	//authenticate with temp password
	user, err = user.Authenticate()
	if err != nil {
		http.Error(res, fmt.Sprintf("%v", err.Error()), http.StatusForbidden)
		return
	}

	log.Debug.Printf("Authenticated inactive user %s", user.Name)

	//set the real password and clear the temp password in the database
	pwHash := sha1.New()
	io.WriteString(pwHash, newPassword)
	user.Password = fmt.Sprintf("%x", pwHash.Sum(nil))
	user.TempPassword = ""
	user.TempExpiry = time.Now()
	user, err = user.update()
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to activate: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	//changed the password successfully,
	//now create session - same as login
	s, err := Session{}.Create(user)
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to create session: %v", err.Error()), http.StatusBadRequest)
		return
	}

	log.Info.Printf("Logged in %s with session %s", user.Name, s.ID.Hex())
	jsonData, err := json.Marshal(s)
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to encode response: %v", err.Error()), http.StatusBadRequest)
		return
	}
	res.Write(jsonData)
} //activateHandler()

func loginHandler(res http.ResponseWriter, req *http.Request) {
	//request data either POSTed or in GET URL
	user := User{}
	if req.Method == http.MethodGet {
		user.Name = req.URL.Query().Get("name")
		user.Password = req.URL.Query().Get("password")
	} else {
		jsonDecoder := json.NewDecoder(req.Body)
		if err := jsonDecoder.Decode(&user); err != nil {
			http.Error(res, fmt.Sprintf("Invalid JSON: %v", err.Error()), http.StatusBadRequest)
			return
		}
	}
	log.Debug.Printf("Login: %+v", user)

	//authenticate with specified password
	//(reset temp in case it was specified)
	user.TempPassword = ""
	var err error
	user, err = user.Authenticate()
	if err != nil {
		http.Error(res, fmt.Sprintf("%v", err.Error()), http.StatusForbidden)
		return
	}

	log.Debug.Printf("Authenticated active user %s", user.Name)

	//now create session - same as login
	s, err := Session{}.Create(user)
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to create session: %v", err.Error()), http.StatusBadRequest)
		return
	}

	log.Info.Printf("Logged in %s with session %s", user.Name, s.ID.Hex())
	jsonData, err := json.Marshal(s)
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to encode response: %v", err.Error()), http.StatusBadRequest)
		return
	}
	res.Write(jsonData)
} //loginHandler()

func logoutHandler(res http.ResponseWriter, req *http.Request) {
	session := Session{}
	if req.Method == http.MethodGet {
		session.ID = bson.ObjectIdHex(req.URL.Query().Get("id"))
	} else {
		jsonDecoder := json.NewDecoder(req.Body)
		if err := jsonDecoder.Decode(&session); err != nil {
			http.Error(res, fmt.Sprintf("Invalid JSON: %v", err.Error()), http.StatusBadRequest)
			return
		}
	}
	log.Debug.Printf("Logout: %+v", session)
	session, err := session.Verify()
	if err != nil {
		http.Error(res, fmt.Sprintf("Unknown session"), http.StatusBadRequest)
		return
	}
	//end the session
	log.Debug.Printf("Ending session %+v", session)
	session.End()
} //logoutHandler()
