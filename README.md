# Session Sign Up Service

![Coverage](https://img.shields.io/badge/Coverage-61.3%25-yellow)

When someone signs up for an Info Session on [operationspark.org](https://operationspark.org),
this service runs a series of tasks:

- Sends a webhook to Greenlight
- Sends a Slack message to the [#signups](https://operationspark.slack.com/archives/G3F2KFGJH) channel.
- Sends the user a confirmation email
- Registers the user for the Info Session's Zoom meeting

## Development

Google provides a [framework](https://cloud.google.com/functions/docs/functions-framework) to run the serverless functions locally. The framework starts an HTTP server that wraps the serverless function(s). You can start the local server with the terminal or VS Code

### Shell

```shell
$ cd cmd
$ SLACK_WEBHOOK_URL=[webhook endpoint] go run main.go

Serving function:
```

Then trigger the function with an HTTP request (cURL, Postman, etc)

```shell
$ curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"firstName":"Quinta", "lastName": "Brunson"}' \
  http://localhost:8080/
```

### VS Code

Use the "Local Function Server" debug configuration:

<img width="600" alt="image" src="https://user-images.githubusercontent.com/9354822/155805725-4de75940-d788-4265-a6cd-a42145e197bb.png">

### Project Structure

There is a `signupService` struct that has a list of `tasks` to run on a [signup form](https://operationspark.org/programs/workforce/infoSession) event from the operationspark.org website.

A `task` is an [interface](https://go.dev/tour/methods/9) with `run` and `name` methods.

```go
type task interface {
	// Run takes a signup form struct and executes some action.
	// Ex.: Send an email, post a Slack message.
	run(context.Context, Signup) error
	// Name Returns the name of the task.
	name() string
}
```

When someone signs up for an Info Session, the form is parsed, then passed to a series of tasks for processing.

```go
// function.go

// Set up services/tasks to run when someone signs up for an Info Session.
mgSvc := NewMailgunService(mgDomain, mgAPIKey, "")
glSvc := NewGreenlightService(glWebhookURL)
slackSvc := NewSlackService(slackWebhookURL)

// These registration tasks include:
registrationService := newSignupService(
		signupServiceOptions{
			// Registration tasks:
			// (executed serially)
			tasks: []task{
				// posting a WebHook to Greenlight,
				glSvc,
				// sending a "Welcome Email",
				mgSvc,
				// sending a Slack message to #signups channel,
				slackSvc,
				// registering the user for the Zoom meeting,
				zoomSvc,
			},
		},
)

server := newSignupServer(registrationService)
```

```go
// Register executes a series of tasks in order. If one fails, the remaining tasks are cancelled.
func (sc *SignupService) register(su Signup) error {
	for _, task := range sc.tasks {
		err := task.run(su)
		if err != nil {
			return fmt.Errorf("task failed: %q: %v", task.name(), err)
		}
	}
	return nil
}
```

To register a new `task`, create a service struct and implement the `task` interface:

#### Example

```go
package signup

type skywriteService struct {
}

func NewSkyWriteService() *skywriteService {
  return &skywriteService{}
}

func (d skywriteService) run(su Signup) error {
	return d.skyWrite(su.NameFirst)
}

func (d skywriteService) name() string {
	return "dominos service"
}

// SkyWrite sends a drone out to draw someones name in chemtrails.
func (d skywriteService) skyWrite(name string) error {
	// Do the thing
	return nil
}
```

Then pass the service to the registration service in `function.go`.

```go
func NewServer() {
	// ...
  registrationService := newSignupService(
    // ... other tasks,
    // (Order matters!)
    NewSkyWriteService()
	)
  // ...
	server := newSignupServer(registrationService)
	return server
}
```

## Connected Services

- [OS Signups App](https://operationspark.slack.com/apps/A0338E8UFFV-os-signups?tab=settings&next_id=0)
- Greenlight Signup API
- Mailgun API

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.
