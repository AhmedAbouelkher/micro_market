# MICRO MARKET

## About the project

The project goal is to show that I understand the underline principles of OpenTelemetry and observability in general.

This project I will present in a technical interview at Dynatrace for the position: <https://www.dynatrace.com/careers/jobs/1377796300/> .

## Headlines and content

The goal is to write a simple but complete documentations for the current project, more like a simple blog post discussing the following:

- General Idea about the project.
  - using microservices architecture.
  - using OpenTelemetry for tracing find problems quickly.
  - using OpenTelemetry for logging and metrics.
- The motivation behind the project.
- brief description of each service.
- brief description of the project directories and files structure.
- The communications between services using sequence diagrams or any other necessary diagrams.
- Discuss tools used and why (the why part should be very brief):
  - golang packages.
  - Grafana docker-otel-lgtm: <https://github.com/grafana/docker-otel-lgtm/>
  - Docker compose for quick and simple deployments @docker-compose.yml
  - kubernetes and the use of kind: <https://github.com/kubernetes-sigs/kind/> @scripts/k8s-up.sh and @scripts/k8s-down.sh and @scripts/port-forward.sh @k8s
  - gRPC for the communication between micro services @proto
  - Load generator to simulate user behavior entreating with the services. @cmd/load-generator @scripts/run_load-generator.sh

## Running

To run and play with the project you have 3 options:

- Using locally using:
  - `INVENTORY_SERVICE_ADDRESS=localhost:9090 GRPC_PORT=8080 HTTP_PORT=8888 make run_checkout`
  - `CHECKOUT_SERVICE_ADDRESS=localhost:8080 GRPC_PORT=9090 HTTP_PORT=9999 make run_inventory``
  - `docker run -p 3000:3000 -p 4317:4317 -p 4318:4318 --rm -ti grafana/otel-lgtm`
- Using docker compose (docker must be installed on your machine):
  - Take a look at @docker-compose.example.yml and use your own external collector configs. Using an external collector is totally optional.
  - `docker compose up --build`
  - `docker compose down`
- Using Kubernetes (kubectl and kind must be installed on your machine & make sure that scripts are executable)
  - Take a look at @k8s/secrets.example.yaml and use your own external collector configs. Using an external collector is totally optional.
  - `chmod +x ./scripts/*.sh`
  - `./scripts/k8s-up.sh`
  - `./scripts/k8s-down.sh`

## AI Usage

I used AI in this project for the following roles:

- Research
- Generate internal tools like the load-generator @cmd/load-generator and scripts @scripts.
- Implementing some of the features:
  - Microservices docker image creation and docker compose deployment.
  - Kubernetes cluster configurations and scripts.
- Improving the Makefile @Makefile to deliver more comfortable DX (Developer Experience)

## Resources

- <https://github.com/open-telemetry/opentelemetry-demo> and <https://www.dynatrace.com/news/blog/opentelemetry-demo-application-with-dynatrace/>
- <https://opentelemetry.io/docs/languages/go/instrumentation/>
- <https://opentelemetry.io/docs/collector/>
- <https://grpc.io/docs/languages/go/basics/>
- <https://www.lucavall.in/blog/opentelemetry-a-guide-to-observability-with-go>
- <https://docs.dynatrace.com/docs/ingest-from/opentelemetry/otlp-api>

## Output

You should generate/update the @README.md file which acts as the central docs for the whole project. Use Table of contents section and in each section add a simple (Back to contents) button to make it easy to navigate the project docs.
