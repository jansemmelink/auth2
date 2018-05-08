package item

import (
	"fmt"
	"net/smtp"
)

//SendMail sends email through mail.google.com
func SendMail(toEmail string, subject string, htmlMessage string) error {
	boundary := "boundary-type-1234567890-alt"
	var msgString string
	//header
	msgString = msgString + "To: " + toEmail + "\r\n"
	msgString = msgString + "From: \"TrotsEk\" <trotsek@gmail.com>\r\n"
	msgString = msgString + "Subject: " + subject + "\r\n"
	msgString = msgString + "MIME-Version: 1.0\r\n"
	msgString = msgString + "Content-Type: multipart/alternative;boundary=\"" + boundary + "\"\r\n"
	msgString = msgString + "\r\n"

	//message part
	/*msgString = msgString + "--" + boundary + "\r\n"
	msgString = msgString + "Content-Type: text/plain; charset=\"iso-8859-1\"\r\n"
	msgString = msgString + "Content-Transfer-Encoding: quoted-printable\r\n"
	msgString = msgString + "\r\n"
	msgString = msgString + message + "\r\n"
	msgString = msgString + "\r\n"*/

	//message part
	msgString = msgString + "--" + boundary + "\r\n"
	msgString = msgString + "Content-Type: text/html; charset=ISO-8859-1\r\n"
	msgString = msgString + "\r\n"
	msgString = msgString + htmlMessage + "\r\n"
	msgString = msgString + "\r\n"

	log.Debug.Printf("HTML: %s", htmlMessage)

	//terminate list of message parts
	msgString = msgString + "--" + boundary + "--\r\n"

	msgBytes := []byte(msgString)
	auth := smtp.PlainAuth("", "jan.semmelink@gmail.com", "tbconiwnujmhnoha", "smtp.gmail.com")
	var to []string
	to = append(to, toEmail)
	err := smtp.SendMail("smtp.gmail.com:587", auth, "jan.semmelink@gmail.com", to, msgBytes)
	if err != nil {
		return fmt.Errorf("Failed to send mail: %s", err.Error())
	}
	return nil
} //SendMail()

/*
From: "Edited Out" <editedout@yahoo.com>
To: "Edited Out" <editedout@yahoo.com>
Subject: Testing 4
MIME-Version: 1.0
Content-Type: multipart/alternative;
  boundary="boundary-type-1234567892-alt"

--boundary-type-1234567892-alt
Content-Type: text/plain; charset="iso-8859-1"
Content-Transfer-Encoding: quoted-printable


Testing the text to see if it works!

--boundary-type-1234567892-alt
Content-Type: text/html; charset="iso-8859-1"
Content-Transfer-Encoding: quoted-printable


<html>Does this actually work?</html>

--boundary-type-1234567892-alt
Content-Transfer-Encoding: base64
Content-Type: text/plain;name="Here2.txt"
Content-Disposition: attachment;filename="Here2.txt"

KiAxMyBGRVRDSCAoQk9EWVtURVhUXSB7NjU5fQ0KLS1fZjZiM2I1ZWUtMjA3YS00ZDdiLTg0NTgtNDY5YmVlNDkxOGRhXw0    KQ29udGVudC1UeXBlOiB0ZXh0L3BsYWluOyBjaGFyc2V0PSJpc28tODg1OS0xIg0KQ29udGVudC1UcmFuc2Zlci1FbmNvZG    luZzogcXVvdGVkLXByaW50YWJsZQ0KDQoNCkp1c3Qgc2VlaW5nIHdoYXQgdGhpcyBhY3R1
YWxseSBjb250YWlucyEgCQkgCSAgIAkJICA9DQoNCi0tX2Y2YjNiNWVlLTIwN2EtNGQ3Yi04NDU4LTQ2OWJlZTQ5MThkYV8    NCkNvbnRlbnQtVHlwZTogdGV4dC9odG1sOyBjaGFyc2V0PSJpc28tODg1OS0xIg0KQ29udGVudC1UcmFuc2Zlci1FbmNvZG    luZzogcXVvdGVkLXByaW50YWJsZQ0KDQo8aHRtbD4NCjxoZWFkPg0KPHN0eWxlPjwhLS0N
Ci5obW1lc3NhZ2UgUA0Kew0KbWFyZ2luOjBweD0zQg0KcGFkZGluZzowcHgNCn0NCmJvZHkuaG1tZXNzYWdlDQp7DQpmb25    0LXNpemU6IDEwcHQ9M0INCmZvbnQtZmFtaWx5OlRhaG9tYQ0KfQ0KLS0+PC9zdHlsZT48L2hlYWQ+DQo8Ym9keSBjbGFzcz    0zRCdobW1lc3NhZ2UnPjxkaXYgZGlyPTNEJ2x0cic+DQpKdXN0IHNlZWluZyB3aGF0IHRo
aXMgYWN0dWFsbHkgY29udGFpbnMhIAkJIAkgICAJCSAgPC9kaXY+PC9ib2R5Pg0KPC9odG1sPj0NCg0KLS1fZjZiM2I1ZWU    tMjA3YS00ZDdiLTg0NTgtNDY5YmVlNDkxOGRhXy0tDQopDQpmbHlubmNvbXB1dGVyIE9LIEZFVENIIGNvbXBsZXRlZA


--boundary-type-1234567890-alt--
*/
