package titan

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/neptulon/neptulon"
)

type gProfile struct {
	Name    string
	Email   string
	Picture []byte
}

type tokenContainer struct {
	Token string `json:"token"`
}

// googleAuth authenticates a user with Google+ using provided OAuth 2.0 access token.
// If authenticated successfully, user profile is retrieved from Google+ and user is given a JWT token in return.
func googleAuth(ctx *neptulon.ReqCtx, db DB, pass string) error {
	var r tokenContainer
	if err := ctx.Params(&r); err != nil || r.Token == "" {
		ctx.Err = &neptulon.ResError{Code: 666, Message: "Malformed or null Google oauth access token was provided."}
		return fmt.Errorf("middleware: auth: google: malformed or null Google oauth token '%v' was provided: %v", r.Token, err)
	}

	p, err := getGProfile(r.Token)
	if err != nil {
		ctx.Err = &neptulon.ResError{Code: 666, Message: "Failed to authenticated with the given Google oauth access token."}
		return fmt.Errorf("middleware: auth: google: error during Google+ profile call using provided access token: %v with error: %v", r.Token, err)
	}

	// retrieve user information
	user, ok := db.GetByMail(p.Email)
	if !ok {
		// this is a first-time registration so create user profile via Google+ profile info
		user = &User{Email: p.Email, Name: p.Name, Picture: p.Picture}

		// save the user information for user ID to be generated by the database
		if err := db.SaveUser(user); err != nil {
			return fmt.Errorf("middleware: auth: google: failed to persist user information: %v", err)
		}

		// create the JWT token
		token := jwt.New(jwt.SigningMethodHS256)
		token.Claims["userid"] = user.ID
		token.Claims["created"] = time.Now().Unix()
		user.JWTToken, err = token.SignedString([]byte(pass))
		if err != nil {
			return fmt.Errorf("middleware: auth: google: jwt signing error: %v", err)
		}

		// now save the full user info
		if err := db.SaveUser(user); err != nil {
			return fmt.Errorf("middleware: auth: google: failed to persist user information: %v", err)
		}
	}

	ctx.Res = tokenContainer{Token: user.JWTToken}
	return nil
}

// ################ Google OAuth2 TokenInfo API Call ################

// verifyIDToken verifies and returns ID token info as described in:
// https://developers.google.com/identity/sign-in/android/backend-auth#send-the-id-token-to-your-server
func getTokenInfo(idToken string) (profile *gProfile, err error) {
	uri := fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=%s", idToken)
	res, err := http.Get(uri)
	if err != nil || res.StatusCode >= http.StatusBadRequest {
		err = fmt.Errorf("failed to call google oauth2 api with error: %v, and response: %+v", err, res)
		return
	}

	resBody, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		err = fmt.Errorf("failed to read response from google oauth2 api with error: %v", err)
		return
	}

	var ti gTokenInfo
	if err = json.Unmarshal(resBody, &ti); err != nil {
		err = fmt.Errorf("failed to deserialize google oauth2 api response with error: %v", err)
		return
	}

	// check that 'aud' claim contains our client id
	if ti.AUD != gServerClient {
		err = fmt.Errorf("given google oauth2 id token belongs to another app id: %v", ti.AUD)
		return
	}

	// retrieve profile image
	uri = ti.Picture
	res, err = http.Get(uri)
	if err != nil {
		err = fmt.Errorf("failed to call google oauth2 api to get user image with error: %v", err)
		return
	}

	profilePic, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		err = fmt.Errorf("failed to read google oauth2 api user profile pic response with error: %v", err)
		return
	}

	profile = &gProfile{Name: ti.GivenName + " " + ti.FamilyName, Email: ti.Email, Picture: profilePic}
	return
}

var gServerClient = "218602439235-6g09g0ap6i8v25v3rel49rtqjcu9ppj0.apps.googleusercontent.com"

type gTokenInfo struct {
	ISS string
	SUB string
	AZP string
	AUD string
	IAT string
	EXP string

	Email         string
	EmailVerified bool `json:"email_verified"`
	Name          string
	Picture       string // profile pic URL
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string
}

// ################ Google+ API Call ################

// getGProfile retrieves user info (display name, e-mail, profile pic) using an oauth2 access token that has 'profile' and 'email' scopes.
// Also retrieves user profile image via profile image URL provided the response.
func getGProfile(oauthToken string) (profile *gProfile, err error) {
	// retrieve profile info from Google
	uri := fmt.Sprintf("https://www.googleapis.com/plus/v1/people/me?access_token=%s", oauthToken)
	res, err := http.Get(uri)
	if err != nil || res.StatusCode >= http.StatusBadRequest {
		err = fmt.Errorf("failed to call google+ api with error: %v, and response: %+v", err, res)
		return
	}

	resBody, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		err = fmt.Errorf("failed to read response from google+ api with error: %v", err)
		return
	}

	var p gPlusProfile
	if err = json.Unmarshal(resBody, &p); err != nil {
		err = fmt.Errorf("failed to deserialize google+ api response with error: %v", err)
		return
	}

	// retrieve profile image
	uri = p.Image.URL
	res, err = http.Get(uri)
	if err != nil {
		err = fmt.Errorf("failed to call google+ api to get user image with error: %v", err)
		return
	}

	profilePic, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		err = fmt.Errorf("failed to read google+ api user profile pic response with error: %v", err)
		return
	}

	profile = &gProfile{Name: p.DisplayName, Email: p.Emails[0].Value, Picture: profilePic}
	return
}

// Response from GET https://www.googleapis.com/plus/v1/people/me?access_token=... (with scope 'profile' and 'email')
// has the following structure with denoted fields of interest (rest is left out):
type gPlusProfile struct {
	Emails      []gPlusEmail
	DisplayName string
	Image       gPlusImage
}

type gPlusEmail struct {
	Value string
}

type gPlusImage struct {
	URL string
}
