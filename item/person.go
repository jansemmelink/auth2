package item

import (
	"path"
	"runtime"

	"gopkg.in/mgo.v2/bson"
)

//Person is stored do describe a real person
type Person struct {
	ID bson.ObjectId `bson:"_id" json:"_id"`

	//a person may have a user id if the person registered for login
	//else this will be undefined ""
	UserID bson.ObjectId `bson:"_user_id" json:"_user_id"`

	//profile
	Names []string
}

var (
	//regexValidName   = regexp.MustCompile("^[a-zA-Z][0-9a-zA-Z]*$")
	dbPersonCollection = Db().DB("item").C("persons")
)

//Validate ...
func (notUsedItem Person) Validate() error {
	//Name
	person := notUsedItem

	if len(person.Names) == 0 {
		return log.Errorf(nil, "Missing names")
	}
	for _, name := range person.Names {
		if name == "" {
			return log.Errorf(nil, "Names may not have empty string values")
		}
	}

	/*_, err := mail.ParseAddress(u.Email)
	if err != nil {
		return log.Errorf(nil, "Invalid email address: \"%s\"", u.Email)
	}*/

	log.Debug.Printf("Validated Person=%+v", person)
	return nil
} //Person.Validate()

//Blank ...
func (notUsedItem Person) Blank() interface{} {
	return Person{}
}

//New ...
func (notUsedItem Person) New(newDataInterfacePtr interface{}) (interface{}, error) {
	//convert type
	var ok bool
	var newDataPtr *Person
	if newDataPtr, ok = newDataInterfacePtr.(*Person); !ok {
		return "", log.Errorf(nil, "Invalid data, got %T but expected (*item.Person)", newDataInterfacePtr)
	}

	p := *newDataPtr
	if err := p.Validate(); err != nil {
		return "", log.Errorf(err, "Invalid person data")
	}

	//TODO: check all unique keys, e.g. all national ids must be unique
	/*
		if _, err := GetPersonByEmail(u.Email); err == nil {
			return "", log.Errorf(err, "Person %s already exists", u.Email)
		}*/

	//assign ID and save
	p.ID = bson.NewObjectId()
	if !p.UserID.Valid() {
		p.UserID = bson.ObjectIdHex("FFFFFFFFFFFFFFFFFFFFFFFF")
	}

	log.Debug.Printf("Creating id=%v", p.ID)
	err := dbPersonCollection.Insert(p)
	if err != nil {
		return "", log.Errorf(err, "Failed to db.insert(%+v)", p)
	}
	log.Info.Printf("Created(%+v)", p)
	return p, nil //.ID.Hex(), nil
} //Person.New()

//Get ...
func (notUsedItem Person) Get(id string) (interface{}, error) {
	log.Debug.Printf("Getting id=%s", id)
	if !bson.IsObjectIdHex(id) {
		return Person{}, log.Errorf(nil, "Invalid id='%s' is not bson hex object id", id)
	}
	mgoKey := make(bson.M)
	mgoKey["_id"] = bson.ObjectIdHex(id)
	personData := Person{}
	if err := dbPersonCollection.Find(mgoKey).One(&personData); err != nil {
		return Person{}, log.Errorf(err, "Failed to get id=%s", id)
	}
	return personData, nil
} //Person.Get()

//GetKey to get by any key fields
func (notUsedItem Person) GetKey(key map[string]string) (interface{}, error) {
	log.Debug.Printf("Getting key=%+v", key)
	mgoKey := make(bson.M)
	for n, v := range key {
		if n == "id" {
			mgoKey["_id"] = bson.ObjectIdHex(v)
		} else {
			mgoKey[n] = v
		}
	}
	personData := Person{}
	if err := dbPersonCollection.Find(mgoKey).One(&personData); err != nil {
		return Person{}, log.Errorf(err, "Failed to get %+v", key)
	}
	return personData, nil
} //Person.GetKey()

//GetPersonByEmail ...
func GetPersonByEmail(email string) (Person, error) {
	log.Debug.Printf("Getting email=%s", email)
	mgoKey := make(bson.M)
	mgoKey["email"] = email
	u := Person{}
	if err := dbPersonCollection.Find(mgoKey).One(&u); err != nil {
		return Person{}, log.Errorf(err, "Person(email=%s) does not exist", email)
	}
	return u, nil
} //GetPersonByEmail()

//GetPersonAuth is called from login operation to load person only with matching credentials
func GetPersonAuth(email, pwSha1 string) (Person, error) {
	log.Debug.Printf("Getting email=%s,pwsha1=%s", email, pwSha1)
	mgoKey := make(bson.M)
	mgoKey["email"] = email
	mgoKey["passwordsha1"] = pwSha1
	u := Person{}
	if err := dbPersonCollection.Find(mgoKey).One(&u); err != nil {
		return Person{}, log.Errorf(nil, "Invalid credentials")
	}
	return u, nil
} //GetPersonAuth()

//Upd ...
func (notUsedItem Person) Upd(id string, newDataInterfacePtr interface{}) error {
	//convert type
	var ok bool
	var newDataPtr *Person
	if newDataPtr, ok = newDataInterfacePtr.(*Person); !ok {
		return log.Errorf(nil, "Invalid person data, got %T but expected (*auth.Person)", newDataInterfacePtr)
	}

	u := *newDataPtr
	if err := u.Validate(); err != nil {
		return log.Errorf(err, "Invalid person data")
	}

	//id specified on URL and id in body contents must be the same
	if u.ID.Hex() != id {
		return log.Errorf(nil, "Specified id=%s not same as data id=%s", id, u.ID.Hex())
	}

	log.Debug.Printf("Updating id=%v", u.ID)
	err := dbPersonCollection.UpdateId(u.ID, u)
	if err != nil {
		return log.Errorf(err, "Failed to db.update(%+v)", u)
	}
	log.Info.Printf("Updated(%+v)", u)
	return nil
} //Person.Upd()

//Del ...
func (notUsedItem Person) Del(id string) error {
	log.Debug.Printf("Deleting id=%s...", id)
	err := dbPersonCollection.RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		return log.Errorf(err, "Failed to delete id=%+v from mongo", id)
	}
	log.Debug.Printf("Deleted id=%s", id)
	return nil
} //Person.Del()

/*//NewPerson creates a new person in memory
func NewPerson(email string, passwordSha1 string) Person {
	return Person{
		Email:        email,
		PasswordSha1: passwordSha1,
	}
}*/ //NewPerson()

// staticDir builds a full path to the 'static' directory
// that is relative to this file.
func templatesDir() string {

	// Locate from the runtime the location of
	// the apps static files.
	_, filename, _, _ := runtime.Caller(1)

	// Return a path to the static folder.
	return path.Join(path.Dir(filename), "templates")
}
