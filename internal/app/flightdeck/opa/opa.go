package opa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	m "github.com/byuoitav/auth/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

func (client *Client) Authorize(log *zap.Logger) gin.HandlerFunc {
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
		if user, ok := c.Request.Context().Value("user").(string); ok {
			opaData.Input.User = user
			log.Debug("User Found\n")
			log.Debug(fmt.Sprintf("Username: %s\n", c.Request.Context().Value("user").(string)))
		} else if apiKey, ok := m.GetAVAPIKey(c.Request.Context()); ok {
			opaData.Input.APIKey = apiKey
		}

		// Prep the request
		oReq, err := json.Marshal(opaData)
		if err != nil {
			log.Error(fmt.Sprintf("Error trying to create request to OPA: %s\n", err))
			c.JSON(http.StatusInternalServerError, gin.H{"Error contacting authorization server": err.Error()})
		}

		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/v1/data/flightdeck", client.URL),
			bytes.NewReader(oReq),
		)

		log.Debug(fmt.Sprintf("%s/v1/data/flightdeck\n", client.URL))

		req.Header.Set("authorization", fmt.Sprintf("Bearer %s", client.Token))

		log.Debug(fmt.Sprintf("Data: %s\n", opaData))
		log.Debug(fmt.Sprintf("URL: %s\n", client.URL))

		// Make the request
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error(fmt.Sprintf("Error while making request to OPA: %s", err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		if res.StatusCode != http.StatusOK {
			log.Error(fmt.Sprintf("Got back non 200 status from OPA: %d", res.StatusCode))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"Authorization server returned non 200 status": err.Error()})
		}

		// Read the body
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error(fmt.Sprintf("Unable to read body from OPA: %s", err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"Authorization server returned non 200 status": err.Error()})
		}

		log.Debug(fmt.Sprintf("Body Output: %s", body))

		// Unmarshal the body
		oRes := opaResponse{}
		err = json.Unmarshal(body, &oRes)
		if err != nil {
			fmt.Errorf("Unable to parse body from OPA: %s", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"Unable to parse body from OPA Server": err.Error()})
		}

		// If OPA approved then allow the request, else reject with a 403
		if oRes.Result.Allow {
			c.Next()
			return
		} else {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ServerStatus": oRes.Result.Allow})
		}
	}
}
