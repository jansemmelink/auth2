package auth

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

//Session represents a user session after login
type Session struct {
	//public: stored in DB
	ID        bson.ObjectId `bson:"_id" json:"id"`
	UserID    bson.ObjectId `bson:"user_id" json:"user_id"`
	StartTime time.Time
	LastTime  time.Time
	Ended     bool

	//private: not stored in DB
	//user User
}

var (
	dbSessionCollection = Db().DB("auth").C("sessions")
)

//Create is called from activate/login operation
//to create a session for the already authenticated user
func (s Session) Create(u User) (Session, error) {
	//TODO: Limit nr of sessions per user, or close old sessions before creating a new one

	//describe the new session
	s.ID = bson.NewObjectId()
	s.UserID = u.ID
	s.StartTime = time.Now()
	s.LastTime = time.Now()
	s.Ended = false

	//create it in the database
	err := dbSessionCollection.Insert(s)
	if err != nil {
		return Session{}, log.Errorf(err, "Failed to db.insert(%+v)", s)
	}
	log.Info.Printf("Session Started: %+v", s)
	return s, nil
} //CreateSession()

//Verify loads the latest session data from s.ID from the db
func (s Session) Verify() (Session, error) {
	if !bson.IsObjectIdHex(s.ID.Hex()) {
		return Session{}, log.Errorf(nil, "Invalid session id='%s'", s.ID.Hex())
	}
	mgoKey := make(bson.M)
	mgoKey["_id"] = s.ID
	sessionData := Session{}
	if err := dbSessionCollection.Find(mgoKey).One(&sessionData); err != nil {
		return Session{}, log.Errorf(nil, "Session.id=%s does not exist", s.ID.Hex())
	}
	if sessionData.Ended {
		return Session{}, log.Errorf(nil, "Session.id=%s already ended", s.ID.Hex())
	}
	sessionExpiry := sessionData.LastTime.Add(time.Minute * 10)
	if time.Now().After(sessionExpiry) {
		return Session{}, log.Errorf(nil, "Session.id=%s expired", s.ID.Hex())
	}
	return sessionData, nil
} //Session.Verify()

//Update ...
func (s *Session) Update() error {
	if s.ID == "" {
		return log.Errorf(nil, "Session update without ID")
	}
	s.LastTime = time.Now()
	err := dbSessionCollection.UpdateId(s.ID, s)
	if err != nil {
		return log.Errorf(err, "Failed to db.update(%+v)", s)
	}
	log.Info.Printf("Updated(%+v)", s)
	return nil
} //Session.Update()

//End is called to end the session
func (s *Session) End() error {
	//verify the session exists
	var verifiedSession Session
	var err error
	if verifiedSession, err = s.Verify(); err != nil {
		return log.Errorf(nil, "Session cannot be verified")
	}

	log.Debug.Printf("Verified session=%+v", verifiedSession)
	//verify: session has not already ended
	if verifiedSession.Ended {
		return log.Errorf(nil, "Session(%s) already ended at %v", verifiedSession.ID, verifiedSession.LastTime)
	}

	//end the session
	verifiedSession.Ended = true
	if err := verifiedSession.Update(); err != nil {
		return err
	}

	//clear ID so cannot update again after this
	*s = verifiedSession
	s.ID = ""
	log.Info.Printf("Session Ended: %+v", s)
	return nil
} //Session.End()

//GetSession ...
func GetSession(sid string) (Session, error) {
	s := Session{
		ID: bson.ObjectIdHex(sid),
	}
	log.Debug.Printf("Looking for session=%+v", s)
	var err error
	if s, err = s.Verify(); err != nil {
		log.Error.Printf("Could not verify session=%+v", s)
		return Session{}, log.Errorf(err, "Invalid session")
	}
	log.Debug.Printf("Verified session=%+v", s)
	return s, nil
}
