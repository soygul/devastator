package test

import "testing"

func TestAuth(t *testing.T) {
	// t.Fatal("Unauthorized clients cannot call any function other than method.auth and method.close") // call to randomized and all registered routes here
	// t.Fatal("Anonymous calls to method.auth and method.close should be allowed")
}

func TestGoogleAuth(t *testing.T) {
	// t.Fatal("Google+ first sign-in (registration) failed with valid credentials")
	// t.Fatal("Google+ second sign-in (regular) failed with valid credentials")
	// t.Fatal("Google+ sign-in passed with invalid credentials")
	// t.Fatal("Authentication was not ACKed")
}

func TestValidClientCertAuth(t *testing.T) {
	s := NewServerHelper(t)
	defer s.Stop()
	c := NewClientHelper(t, true)
	defer c.Close()

	id := c.WriteRequest("auth.cert", nil)
	m := c.ReadMsg()

	if m.ID != id && m.Result != "ACK" {
		t.Fatal("Authentication failed with a valid client certificate. Got server response:", m)
	}

	// t.Fatal("Authenticated with invalid/expired client certificate")
	// t.Fatal("Authentication was not ACKed")
}

func TestTokenAuth(t *testing.T) {
	// t.Fatal("Authentication failed with a valid token")
	// t.Fatal("Authenticated with invalid/expired token")
	// t.Fatal("Authentication was not ACKed")
}
