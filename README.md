<h1 align="center">
    <img src="https://raw.githubusercontent.com/ddosify/ddosify/master/assets/ddosify-logo-db.svg#gh-dark-mode-only" alt="Ddosify logo dark" width="336px" /><br />
    <img src="https://raw.githubusercontent.com/ddosify/ddosify/master/assets/ddosify-logo-wb.svg#gh-light-mode-only" alt="Ddosify logo light" width="336px" /><br />
    "Canva" of Observability 
</h1>

<p align="center">
    <a href="https://github.com/ddosify/ddosify/releases" target="_blank"><img src="https://img.shields.io/github/v/release/ddosify/ddosify?style=for-the-badge&logo=github&color=orange" alt="ddosify latest version" /></a>&nbsp;
    <a href="https://github.com/ddosify/ddosify/actions/workflows/test.yml" target="_blank"><img src="https://img.shields.io/github/actions/workflow/status/ddosify/ddosify/test.yml?branch=master&style=for-the-badge&logo=github" alt="ddosify build result" /></a>&nbsp;
    <a href="https://pkg.go.dev/go.ddosify.com/ddosify" target="_blank"><img src="https://img.shields.io/github/go-mod/go-version/ddosify/ddosify?style=for-the-badge&logo=go" alt="golang version" /></a>&nbsp;
    <a href="https://app.codecov.io/gh/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/codecov/c/github/ddosify/ddosify?style=for-the-badge&logo=codecov" alt="go coverage" /></a>&nbsp;
    <a href="https://goreportcard.com/report/github.com/ddosify/ddosify" target="_blank"><img src="https://goreportcard.com/badge/github.com/ddosify/ddosify?style=for-the-badge&logo=go" alt="go report" /></a>&nbsp;
    <a href="https://github.com/ddosify/ddosify/blob/master/LICENSE" target="_blank"><img src="https://img.shields.io/badge/LICENSE-AGPL--3.0-orange?style=for-the-badge&logo=none" alt="ddosify license" /></a>
    <a href="https://discord.gg/9KdnrSUZQg" target="_blank"><img src="https://img.shields.io/discord/898523141788287017?style=for-the-badge&logo=discord&label=DISCORD" alt="ddosify discord server" /></a>
    <a href="https://hub.docker.com/r/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/docker/v/ddosify/ddosify?style=for-the-badge&logo=docker&label=docker&sort=semver" alt="ddosify docker image" /></a>
</p>

<p align="center">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/5ed79d96-aef4-467d-f5d0-e17cc5c3e700/public" alt="Ddosify - Self-Hosted" />
</p>

### Quick Start

```bash
curl -sSL https://raw.githubusercontent.com/ddosify/ddosify/master/selfhosted/install.sh | bash
```

<a href="https://aws.amazon.com/marketplace/pp/prodview-mwvnujtgjedjy" target="_blank"><img src="https://img.shields.io/badge/Available_on_aws_marketplace-FF9900?style=for-the-badge&logo=amazonaws&logoColor=white" alt="ddosify aws marketplace deployment" /></a>&nbsp;

## What is Ddosify?
Ddosify is a magic wand that instantly spots glitches and guarantees the smooth performance of your infrastructure and application while saving you time and money. Ddosify Platform includes Performance Testing and Kubernetes Observability capabilities. It uniquely integrates these two parts and effortlessly spots the performance issues.

Ddosify Stack consists of 4 parts. Those are **Ddosify Engine, Ddosify eBPF Agent (Alaz), Ddosify Self-Hosted, and Ddosify Cloud**.

<p align="center"> 
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/3f995c37-10fb-4b37-c74f-db1685d0df00/public" alt="Ddosify Stack" />
</p>

### :rocket: Ddosify Engine
This is the load engine of Ddosify, written in Golang. Ddosify Self-Hosted and Ddosify Cloud use it on load generation. It is fully open-source and can be used on the CLI as a standalone tool. Ddosify Engine is available via [Docker](https://hub.docker.com/r/ddosify/ddosify), [Docker Extension](https://hub.docker.com/extensions/ddosify/ddosify-docker-extension), [Homebrew Tap](https://github.com/ddosify/ddosify#homebrew-tap-macos-and-linux), and downloadable pre-compiled binaries from the [releases page](https://github.com/ddosify/ddosify/releases/latest) for macOS, Linux, and Windows.

Check out the [Engine Docs](https://github.com/ddosify/ddosify/tree/master/engine_docs) page for more information and usage.

### 🐝 Ddosify eBPF Agent (Alaz)
[Alaz](https://github.com/ddosify/alaz) is an open-source Ddosify eBPF agent that can inspect and collect Kubernetes (K8s) service traffic without the need for code instrumentation, sidecars, or service restarts. Alaz is deployed as a DaemonSet on your Kubernetes cluster. It collects metrics and sends them to Ddosify Cloud or Ddosify Self-Hosted. It also embeds prometheus node-exporter inside. So that you will have visibility on your cluster nodes also.

Check out the [Alaz](https://github.com/ddosify/alaz) repository for more information and usage.

### 🏠 Ddosify Self-Hosted
Ddosify Self-Hosted features a web-based user interface, distributed load generation, and Kubernetes Monitoring capabilities. While it shares many of the same functionalities as Ddosify Cloud, the Self-Hosted version is designed to be deployed within your own infrastructure for enhanced control and customization. There are two versions of it, **Community Edition (CE)** and **Enterprise Edition (EE)**. You can see the differences in the below comparison table.

Check out the [Self-Hosted](https://github.com/ddosify/ddosify/tree/master/selfhosted) page for more information and usage.

### ☁️ Ddosify Cloud
With Ddosify Cloud, anyone can test the performance of backend endpoints, monitor Kubernetes Clusters, and find the bottlenecks in the system. It has a No code UI, insightful charts, service maps, and more features!

Check out [Ddosify Cloud](https://app.ddosify.com/) to instantly find the performance issues on your system.

### ☁️ Ddosify Cloud vs 🏠 Ddosify Self-Hosted EE  vs 🏡 Ddosify Self-Hosted CE
<p align="center">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/b539346a-171d-4acd-e57e-482947c10300/public" alt="Ddosify versus" />

*CE: Community Edition, EE: Enterprise Edition*
</p>

## Observability Features
#### ✅  Service Map
Easily get insights about what is going on in your cluster. <a href="https://docs.ddosify.com/cloud/observability/service-map" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/891d68ac-554f-4828-a0fe-25ce935d1100/public" alt="Ddosify - Service Map Feature" />
</p>

#### ✅  Detailed Insights
Inspect incoming, outgoing traffic, SQL queries, and more. <a href="https://docs.ddosify.com/cloud/observability/service-map" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/38760abe-8090-47b9-23e6-4cfb2db17000/public" alt="Ddosify - Detailed Insights Feature" />
</p>

#### ✅  Metrics Dashboard
The Metric Dashboard provides a straightforward way to observe Node Metrics. <a href="https://docs.ddosify.com/cloud/observability/metrics" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/e736b581-7269-4ec4-9d77-7d375a5e3a00/public" alt="Ddosify - Metrics Dashboard Feature" />
</p>

#### ✅  Find Bottlenecks 
Start a load test and monitor your system all within the same UI. 
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/32697752-ba0c-4c7b-8f59-f34095d6ef00/public" alt="Ddosify - Find Bottlenecks Feature" />
</p>

## Load Testing Features
#### ✅  Parametrization
Use built-in random data generators. <a href="https://docs.ddosify.com/concepts/parameterization" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/4dc3f294-6319-4c2b-a56b-c359276d5e00/public" alt="Ddosify - Parametrization Feature" />
</p>


#### ✅  CSV Data Import
Import test data from CSV and use it in the scenario. <a href="https://docs.ddosify.com/concepts/test-data-import" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/6c769b68-e046-440d-137c-b20dd3518300/public" alt="Ddosify - Test Data Feature" />
</p>

#### ✅  Environments
Store constant values as environment variables. <a href="https://docs.ddosify.com/concepts/environment-variables" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/78c45dca-03de-4cbf-0edc-fb2b8a2e6600/public" alt="Ddosify - Environment Feature" />
</p>

#### ✅  Correlation
Extract variables from earlier phases and pass them on to the following ones. <a href="https://docs.ddosify.com/concepts/correlation" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/7ac98b0f-043b-494f-3c17-7a5436e81400/public" alt="Ddosify - Correlation Feature" />
</p>

#### ✅  Assertion
Verify that the response matches your expectations. <a href="https://docs.ddosify.com/concepts/assertion" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/f2f0df70-8e2b-4308-ca6a-4259274d0400/public" alt="Ddosify - Assertion Feature" />
</p>

#### ✅  Debugging
Analyze request and response data before starting the load test. <a href="https://docs.ddosify.com/concepts/debugging" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/82322e21-2e3a-4284-9643-1c8dacda2400/public" alt="Ddosify - Debugging Feature" />
</p>

#### ✅  Postman Import
Import Postman collections with ease and transform them into load testing scenarios. <a href="https://docs.ddosify.com/concepts/postman-import" target="_blank">More →</a>
<p align="left">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/873bf1e3-07a0-427c-8f32-0791d1728900/public" alt="Ddosify - Postman Import Feature" />
</p>


## About This Repository

This repository includes the source code for the Ddosify Engine. You can access Docker Images for the Ddosify Engine and Self Hosted on <a href="https://hub.docker.com/u/ddosify" target="_blank">Docker Hub</a>.

The [Engine Docs](https://github.com/ddosify/ddosify/tree/master/engine_docs) folder provides information on the installation, usage, and features of the Ddosify Engine. The [Self-Hosted](https://github.com/ddosify/ddosify/tree/master/selfhosted) folder contains installation instructions for the Self-Hosted version. [Ddosify eBPF agent (Alaz)](https://github.com/ddosify/alaz) has its own repository. To learn about the usage of both Self-Hosted and Cloud versions, please refer to the [this documentation](https://docs.ddosify.com/concepts/test-suite).

## Communication

You can join our [Discord Server](https://discord.gg/9KdnrSUZQg) for issues, feature requests, feedbacks or anything else. 

## Disclaimer

Ddosify is created for testing the performance of web applications. Users must be the owner of the target system. Using it for harmful purposes is extremely forbidden. Ddosify team & company is not responsible for its’ usages and consequences.

## License

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
