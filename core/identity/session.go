package identity

import (
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"

	log "github.com/Sirupsen/logrus"
)

const (
	cookieName   = "session"
	callbackPath = "/oauth/callback"
)

//Session is the information about a logged in user
type Session struct {
	Username string
	Expires  time.Time
	Token    *jwt.Token
}

//IsExpired returns true if the session expired, false if not (or if the session is nil)
func (s *Session) IsExpired() (expired bool) {
	expired = !(s != nil && time.Now().Before(s.Expires))
	return
}

//Protect requires users to log using itsyou.online
func Protect(clientID string, clientSecret string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//If this is the callback from itsyou.online
		if r.URL.Path == callbackPath {
			r.ParseForm()
			code := r.FormValue("code")
			token, err := getJWTToken(code, clientID, clientSecret, r)
			if err != nil {
				//TODO: handle more gracefully than this
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			s, err := verifyJWTToken(token)
			if err != nil {
				log.Debugln("Error processing jwt token:", err, "- TOKEN: ", token)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			startSession(w, s)

			//TODO: handle direct links
			http.Redirect(w, r, "/index.php", http.StatusFound)
			return
		}
		cookie, err := r.Cookie(cookieName)
		//No session
		if err == http.ErrNoCookie {
			redirectToOauthLogin(clientID, w, r)
			return
		}
		token := cookie.Value

		s, err := verifyJWTToken(token)
		if err != nil || s.IsExpired() {
			redirectToOauthLogin(clientID, w, r)
			return
		}
		//TODO: put session in the request context
		handler.ServeHTTP(w, r)
		return
	})
}

func redirectToOauthLogin(clientID string, w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse("https://itsyou.online/v1/oauth/authorize")
	q := u.Query()
	q.Add("client_id", clientID)
	q.Add("state", "STATE")
	//TODO: make this request dependent
	q.Add("redirect_uri", "http://"+r.Host+"/oauth/callback")
	q.Add("response_type", "code")
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

//ClearSession deletes the current session
func ClearSession(w http.ResponseWriter) {
	setCookie(cookieName, time.Time{}, w)
}

func startSession(w http.ResponseWriter, s *Session) {
	log.Infoln("TOKEN:", s.Token)
	setCookie(s.Token.Raw, s.Expires, w)
}

func setCookie(code string, expires time.Time, w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:    cookieName,
		Value:   code,
		Path:    "/",
		Expires: expires,
	}
	http.SetCookie(w, &cookie)
}
