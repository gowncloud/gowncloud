package identity

import (
	"context"
	"net/http"
	"net/url"
	"strings"
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
	Username      string
	Expires       time.Time
	Token         *jwt.Token
	Organizations []string
}

//CurrentSession get's the current session from the request context
func CurrentSession(r *http.Request) (s Session) {
	if rawsession := r.Context().Value("session"); rawsession != nil {
		s = rawsession.(Session)
	}
	return
}

//IsExpired returns true if the session expired, false if not (or if the session is nil)
func (s *Session) IsExpired() (expired bool) {
	expired = !(s != nil && time.Now().Before(s.Expires))
	return
}

//AddIdentity add the current user session to the context, it is seperate from Protect to enable an identity aware logger to be inserted between the them
func AddIdentity(handler http.Handler, clientID string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err == nil {
			token := cookie.Value
			s, err := verifyJWTToken(token, clientID)
			if err == nil {
				//Add session to context
				ctx := context.WithValue(r.Context(), "session", *s)
				handler.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
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
				log.Debugln("Error getting a jwt token:", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			// If the user isnt a member of clientID, itsYou.Online seems to return the following token at the moment
			if token == "Unauthorized\n" {
				log.Debug("Rejected login due to invalid jwt")
				rejectString := "Only members of " + clientID + " have access to this gowncloud server"
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(rejectString))
				return
			}

			s, err := verifyJWTToken(token, clientID)
			if err != nil {
				log.Debugln("Error processing jwt token:", err, "- TOKEN: ", token)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			// Check if the user is in the organization defined by clientID or any of its suborganizations
			authorized := false
			for _, org := range s.Organizations {
				if strings.HasPrefix(org, clientID) {
					authorized = true
					break
				}
			}

			if !authorized {
				// TODO: provide a nice page to tell the user they have been rejected
				rejectString := "Only members of " + clientID + " have access to this gowncloud server"
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(rejectString))
				return
			}

			startSession(w, s)

			//TODO: handle direct links
			http.Redirect(w, r, "/index.php", http.StatusFound)
			return
		}
		s := CurrentSession(r)
		if s.Username == "" || s.IsExpired() {
			redirectToOauthLogin(clientID, w, r)
			return
		}
		handler.ServeHTTP(w, r)
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
	q.Add("scope", "user:memberof:"+clientID)
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
		Name:     cookieName,
		Value:    code,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
}
