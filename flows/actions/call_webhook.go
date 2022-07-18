package actions

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"

	"github.com/pkg/errors"
	"golang.org/x/net/http/httpguts"
)

func isValidURL(u string) bool {
	if utf8.RuneCountInString(u) > 2048 {
		return false
	}
	_, err := url.Parse(u)
	return err == nil
}

func init() {
	registerType(TypeCallWebhook, func() flows.Action { return &CallWebhookAction{} })
}

// TypeCallWebhook is the type for the call webhook action
const TypeCallWebhook string = "call_webhook"

// CallWebhookAction can be used to call an external service. The body, header and url fields may be
// templates and will be evaluated at runtime. A [event:webhook_called] event will be created based on
// the results of the HTTP call. If this action has a `result_name`, then additionally it will create
// a new result with that name. The value of the result will be the status code and the category will be
// `Success` or `Failed`. If the webhook returned valid JSON which is less than 10000 bytes, that will be
// accessible through `extra` on the result. The last JSON response from a webhook call in the current
// sprint will additionally be accessible in expressions as `@webhook` regardless of size.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "call_webhook",
//     "method": "GET",
//     "url": "http://localhost:49998/?cmd=success",
//     "headers": {
//       "Authorization": "Token AAFFZZHH"
//     },
//     "result_name": "webhook"
//   }
//
// @action call_webhook
type CallWebhookAction struct {
	baseAction
	onlineAction

	Method     string            `json:"method" validate:"required,http_method"`
	URL        string            `json:"url" validate:"required" engine:"evaluated"`
	Headers    map[string]string `json:"headers,omitempty" engine:"evaluated"`
	Body       string            `json:"body,omitempty" engine:"evaluated"`
	ResultName string            `json:"result_name,omitempty"`
}

// NewCallWebhook creates a new call webhook action
func NewCallWebhook(uuid flows.ActionUUID, method string, url string, headers map[string]string, body string, resultName string) *CallWebhookAction {
	return &CallWebhookAction{
		baseAction: newBaseAction(TypeCallWebhook, uuid),
		Method:     method,
		URL:        url,
		Headers:    headers,
		Body:       body,
		ResultName: resultName,
	}
}

// Validate validates our action is valid
func (a *CallWebhookAction) Validate() error {
	for key := range a.Headers {
		if !httpguts.ValidHeaderFieldName(key) {
			return errors.Errorf("header '%s' is not a valid HTTP header", key)
		}
	}

	return nil
}

// Execute runs this action
func (a *CallWebhookAction) Execute(run flows.Run, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {

	// substitute any variables in our url
	url, err := run.EvaluateTemplate(a.URL)
	if err != nil {
		logEvent(events.NewError(err))
	}

	url = strings.TrimSpace(url) // some servers don't like trailing spaces in HTTP requests

	if url == "" {
		logEvent(events.NewErrorf("webhook URL evaluated to empty string"))
		return nil
	}
	if !isValidURL(url) {
		logEvent(events.NewErrorf("webhook URL evaluated to an invalid URL: '%s'", url))
		return nil
	}

	method := strings.ToUpper(a.Method)
	body := a.Body

	// substitute any body variables
	if body != "" {
		// webhook bodies aren't truncated like other templates
		body, err = run.EvaluateTemplateText(body, nil, false)
		if err != nil {
			logEvent(events.NewError(err))
		}
	}

	return a.call(run, step, url, method, body, logEvent)
}

// Execute runs this action
func (a *CallWebhookAction) call(run flows.Run, step flows.Step, url, method, body string, logEvent flows.EventCallback) error {
	// build our request
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}

	// add the custom headers, substituting any template vars
	for key, value := range a.Headers {
		headerValue, err := run.EvaluateTemplate(value)
		if err != nil {
			logEvent(events.NewError(err))
		}

		req.Header.Add(key, headerValue)
	}

	redisPool := &redis.Pool{
		Wait:        true,              // makes callers wait for a connection
		MaxActive:   5,                 // only open this many concurrent connections at once
		MaxIdle:     2,                 // only keep up to 2 idle
		IdleTimeout: 240 * time.Second, // how long to wait before reaping a connection
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", fmt.Sprintf("%s", ""))
			if err != nil {
				return nil, err
			}

			// switch to the right DB
			_, err = conn.Do("SELECT", strings.TrimLeft("", "/"))
			return conn, err
		},
	}

	conn := redisPool.Get()
	access_token, err := redis.String(conn.Do("GET", "access_token"))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", access_token))

	svc, err := run.Session().Engine().Services().Webhook(run.Session().Assets())
	if err != nil {
		logEvent(events.NewError(err))
		return nil
	}

	call, err := svc.Call(req)

	if err != nil {
		logEvent(events.NewError(err))
	}
	if call != nil {
		a.updateWebhook(run, call)

		status := callStatus(call, err, false)

		logEvent(events.NewWebhookCalled(call, status, ""))

		if a.ResultName != "" {
			a.saveWebhookResult(run, step, a.ResultName, call, status, logEvent)
		}
	}

	return nil
}

// Results enumerates any results generated by this flow object
func (a *CallWebhookAction) Results(include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, webhookCategories))
	}
}

// determines the webhook status from the HTTP status code
func callStatus(call *flows.WebhookCall, err error, isResthook bool) flows.CallStatus {
	if call.Response == nil || err != nil {
		return flows.CallStatusConnectionError
	}
	if isResthook && call.Response.StatusCode == http.StatusGone {
		// https://zapier.com/developer/documentation/v2/rest-hooks/
		return flows.CallStatusSubscriberGone
	}
	if call.Response.StatusCode/100 == 2 {
		return flows.CallStatusSuccess
	}
	return flows.CallStatusResponseError
}
