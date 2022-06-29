# Slack Session Sign Up Service

When someone signs up for an Info Session on [operationspark.org](https://operationspark.org),
this service sends a webhook to Greenlight and a Slack message to the [#signups](https://operationspark.slack.com/archives/G3F2KFGJH) channel.

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


## Deployment

The service is deployed as a Google Cloud Function and trigger by webhooks from operationspark.org

https://console.cloud.google.com/functions/details/us-central1/session-signups?env=gen1&authuser=1&project=operationspark-org

[TODO: Flesh out]

```shell
$ gcloud functions deploy session-signups \
--runtime=go116 \
--trigger-http \
--entry-point HandleSignUp \
--env-vars-file .env.yaml
```

#### session-signups-dev
```
gcloud functions deploy session-signups-dev --runtime=go116 --trigger-http --entry-point HandleSignUp --env-vars-file .env.yaml
```


## Connected Services

- [OS Signups App](https://operationspark.slack.com/apps/A0338E8UFFV-os-signups?tab=settings&next_id=0)
- Greenlight Signup API


## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.
