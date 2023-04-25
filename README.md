<h1 align="center">
    <img src="https://raw.githubusercontent.com/ddosify/ddosify/master/assets/ddosify-logo-db.svg#gh-dark-mode-only" alt="Ddosify logo dark" width="336px" /><br />
    <img src="https://raw.githubusercontent.com/ddosify/ddosify/master/assets/ddosify-logo-wb.svg#gh-light-mode-only" alt="Ddosify logo light" width="336px" /><br />
    Distributed Performance Testing Platform
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

## Ddosify Self-Hosted (Distributed, No-code UI)
<p align="center">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/c6f26a7b-b878-4af7-774e-b0d65935df00/public" alt="Ddosify - Self-Hosted" />
</p>

## Ddosify Engine (Single node, usage on CLI)
<p align="center">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/68e07b5f-22a5-4244-5dc2-9d02bd2c9e00/public" alt="Ddosify - Engine" />
</p>

## What is Ddosify?
Ddosify is a comprehensive performance testing platform, designed specifically to evaluate backend load and latency. It offers three distinct deployment options to cater to various needs: Ddosify Engine, Ddosify Self-Hosted, and Ddosify Cloud.

### :rocket: Ddosify Engine
This is the load engine of Ddosify, written in Golang. It is fully open-source and can be used on the CLI. Ddosify Engine is available via Docker, Docker Extension, Homebrew Tap, and downloadable pre-compiled binaries from the releases page for macOS, Linux, and Windows.

Check out the [Engine Docs](https://github.com/ddosify/ddosify/tree/master/engine_docs) page for more information and usage.

### 🏠 Ddosify Self-Hosted
In contrast to the Engine version, Ddosify Self-Hosted features a web-based user interface and distributed load generation capabilities. While it shares many of the same functionalities as Ddosify Cloud, the Self-Hosted version is designed to be deployed within your own infrastructure for enhanced control and customization. And it's completely Free!

Check out the [Self-Hosted](https://github.com/ddosify/ddosify/tree/master/selfhosted) page for more information and usage.

### ☁️ Ddosify Cloud
Ddosify Cloud enables users to assess backend endpoints' performance through load and latency testing, offering a user-friendly interface, comprehensive charts, extensive geographic targeting options, and additional features for an improved testing experience.

Check out [Ddosify Cloud](https://ddosify.com) to start effortless testing.

### ☁️ Ddosify Cloud vs 🏠 Ddosify Self-Hosted  vs :rocket: Ddosify Engine
<p align="center">
<img src="https://imagedelivery.net/jnIqn6NB1gbMLXIvlYKo5A/7d6b9778-1367-426e-b6e9-5fc8f0d34200/public" alt="Ddosify versus" />
</p>



## Features

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

The [Engine Docs](https://github.com/ddosify/ddosify/tree/master/engine_docs) folder provides information on the installation, usage, and features of the Ddosify Engine. The [Self-Hosted](https://github.com/ddosify/ddosify/tree/master/selfhosted) folder contains installation instructions for the Self-Hosted version. To learn about the usage of both Self-Hosted and Cloud versions, please refer to the [this documentation](https://docs.ddosify.com/concepts/test-suite).

## Communication

You can join our [Discord Server](https://discord.gg/9KdnrSUZQg) for issues, feature requests, feedbacks or anything else. 

## Disclaimer

Ddosify is created for testing the performance of web applications. Users must be the owner of the target system. Using it for harmful purposes is extremely forbidden. Ddosify team & company is not responsible for its’ usages and consequences.

## License

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
