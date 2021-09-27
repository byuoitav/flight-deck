package opa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	m "github.com/byuoitav/auth/middleware"
	"github.com/gin-gonic/gin"
)

type Client struct {
	URL   string
	Token string
}

type opaResponse struct {
	DecisionID string    `json:"decision_id"`
	Result     opaResult `json:"result"`
}

type opaResult struct {
	Allow bool `json:"allow"`
}

type opaRequest struct {
	Input requestData `json:"input"`
}

type requestData struct {
	APIKey string `json:"api_key"`
	User   string `json:"user"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (client *Client) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Initial data
		opaData := opaRequest{
			Input: requestData{
				Path:   c.FullPath(),
				Method: c.Request.Method,
			},
		}

		// use either the user netid for the authorization request or an
		// API key if one was used instead
		if user, ok := c.Request.Context().Value("userBYUID").(string); ok {
			opaData.Input.User = user
			fmt.Printf("User Found")
		} else if apiKey, ok := m.GetAVAPIKey(c.Request.Context()); ok {
			opaData.Input.APIKey = apiKey
		}

		// Prep the request
		oReq, err := json.Marshal(opaData)
		if err != nil {
			fmt.Errorf("Error trying to create request to OPA: %s\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"Error contacting authorization server": err.Error()})
			return
		}

		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/v1/data/flightdeck", client.URL),
			bytes.NewReader(oReq),
		)

		req.Header.Set("authorization", fmt.Sprintf("Bearer %s", client.Token))

		fmt.Printf("Data: %s\n", opaData)
		fmt.Printf("URL: %s\n", client.URL)

		// Make the request
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Errorf("Error while making request to OPA: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if res.StatusCode != http.StatusOK {
			fmt.Errorf("Got back non 200 status from OPA: %d", res.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"Authorization server returned non 200 status": err.Error()})
			return
		}

		// Read the body
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Errorf("Unable to read body from OPA: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"Authorization server returned non 200 status": err.Error()})
			return
		}

		// Unmarshal the body
		oRes := opaResponse{}
		err = json.Unmarshal(body, &oRes)
		if err != nil {
			fmt.Errorf("Unable to parse body from OPA: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"Upable to parse body from OPA Server": err.Error()})
			return
		}

		// If OPA approved then allow the request, else reject with a 403
		if oRes.Result.Allow {
			c.Next()
			return
		} else {
			c.JSON(http.StatusForbidden, gin.H{"ServerStatus": "Forbidden"})
			return
		}
	}
}
