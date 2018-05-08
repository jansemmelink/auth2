package item

import mgo "gopkg.in/mgo.v2"

const mongoURL = "/item"

var _mdMgoSession *mgo.Session

//Db returns current db connection session
func Db() *mgo.Session {
	if _mdMgoSession == nil {
		var err error
		_mdMgoSession, err = mgo.Dial(mongoURL)
		if err != nil {
			panic("Cannot connect to mongo: " + mongoURL + ": " + err.Error())
		}
	}
	return _mdMgoSession
} //Db()
