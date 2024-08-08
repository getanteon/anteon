<div align="center">
    <img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon-logo-db.svg#gh-dark-mode-only" alt="Anteon logo dark" width="336px" /><br />
    <img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon-logo-wb.svg#gh-light-mode-only" alt="Anteon logo light" width="336px" /><br />
</div>

<h3 align="center">Kickstart Kubernetes Monitoring in 1 min - Free up time for DevOps and Coding</h3>

<p align="center">
    <img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon_service_map.png" alt="Anteon Kubernetes Monitoring Service Map" />
    <p align="center">
        <a href="https://github.com/getanteon/anteon/releases" target="_blank"><img src="https://img.shields.io/github/v/release/getanteon/anteon?style=for-the-badge&logo=github&color=orange" alt="anteon latest version" /></a>&nbsp;
        <a href="https://github.com/getanteon/anteon/blob/master/LICENSE" target="_blank"><img src="https://img.shields.io/badge/LICENSE-AGPL--3.0-orange?style=for-the-badge&logo=none" alt="Anteon license" /></a>
        <a href="https://discord.com/invite/9KdnrSUZQg" target="_blank"><img src="https://img.shields.io/discord/898523141788287017?style=for-the-badge&logo=discord&label=DISCORD" alt="Anteon discord server" /></a>
        <a href="https://landscape.cncf.io/?item=observability-and-analysis--observability--anteon" target="_blank"><img src="https://img.shields.io/badge/CNCF%20Landscape-5699C6?style=for-the-badge&logo=cncf&label=cncf" alt="cncf landscape" /></a>
    </p>
    <i>Anteon automatically generates Service Map of your K8s cluster without code instrumentation or sidecars. So you can easily find the bottlenecks in your system. Red lines indicate the high latency between services.</i>
</p>

<h2 align="center">
    <a href="https://demo.getanteon.com/" target="_blank">Live Demo</a> •
    <a href="https://getanteon.com/docs" target="_blank">Documentation</a> •
    <a href="https://discord.com/invite/9KdnrSUZQg" target="_blank">Discord</a>
</h2>

## 🐝 What is Anteon?

**Anteon** (formerly Ddosify) is an [open-source](https://github.com/getanteon/anteon), eBPF-based **Kubernetes Monitoring** and **Performance Testing** platform.

### 🔎 Kubernetes Monitoring

- **Automatic Service Map Creation:** Anteon automatically creates a **service map** of your cluster without code instrumentation or sidecars. So you can easily [find the bottlenecks](https://getanteon.com/docs/kubernetes-monitoring/#finding-bottlenecks) in your system.
- **Performance Insights:** It helps you spot issues like services taking too long to respond or slow SQL queries.
- **Real-Time Metrics:** The platform tracks and displays live data on your cluster instances CPU, memory, disk, and network usage.
- **Ease of Use:** You don't need to change any code, restart services, or add extra components (like sidecars) to get these insights, thanks to the [eBPF based agent (Alaz)](https://github.com/getanteon/alaz).
- **Alerts for Anomalies:** If something unusual, like a sudden increase in CPU usage, happens in your Kubernetes (K8s) cluster, Anteon immediately sends alerts to your Slack.
- **Seamless Integration with Performance Testing:** Performance testing is natively integrated with Kubernetes monitoring for a unified experience.

<p align="center">
<img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon_metrics.png" alt="Anteon Kubernetes Monitoring Metrics" />
<i>Anteon tracks and displays live data on your cluster instances CPU, memory, disk, and network usage.</i>
</p>

### 🔨 Performance Testing

- **Multi-Location Based:** Generate load/performance tests from over 25 countries worldwide. Its available on [Anteon Cloud](https://getanteon.com/).
- **Easy Scenario Builder:** Create test scenarios easily without writing any code.
- **Seamless Integration with Kubernetes Monitoring:** Performance testing is natively integrated with Kubernetes monitoring for a unified experience.
- **Postman Integration:** Import tests directly from Postman, making it convenient for those already using Postman for API development and testing.

<p align="center">
<img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon_performance_testing.png" alt="Anteon Kubernetes Monitoring Metrics" />
<i>Anteon Performance Testing generates load from worldwide with no-code scenario builder.</i>
</p>

## 📚 Documentation

- [🐝 Anteon Stack](https://getanteon.com/docs/stack/)
- [🚀 Getting Started](https://getanteon.com/docs/getting-started/)
- [🔎 Kubernetes Monitoring](https://getanteon.com/docs/kubernetes-monitoring/)
- [🔨 Performance Testing](https://getanteon.com/docs/performance-testing/)

## ✨ About This Repository

This repository includes the source code for the Anteon Load Engine (Ddosify). You can access Docker Images for the Anteon Engine and Self Hosted on <a href="https://hub.docker.com/u/ddosify" target="_blank">Docker Hub</a>. Since Anteon is a Verified Publisher on Docker Hub, there isn't any pull limits.

- [Ddosify documentation](https://github.com/getanteon/anteon/tree/master/ddosify_engine) provides information on the installation, usage, and features of the Anteon Load Engine.
- The [Self-Hosted](https://github.com/getanteon/anteon/tree/master/selfhosted) folder contains installation instructions for the Self-Hosted version.
- [Anteon eBPF agent (Alaz)](https://github.com/getanteon/alaz) has its own repository.

See the [Anteon website](https://getanteon.com/) for more information.

## 🛠️ Contributing

See our [Contribution Guide](./CONTRIBUTING.md) and please follow the [Code of Conduct](./CODE_OF_CONDUCT.md) in all your interactions with the project.

Thanks goes to these wonderful people!

<a href="https://github.com/getanteon/anteon/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=getanteon/anteon" />
</a>

Made with [contrib.rocks](https://contrib.rocks).

### 📨 Communication

You can join our [Discord Server](https://discord.com/invite/9KdnrSUZQg) for issues, feature requests, feedbacks or anything else.

### ⚠️ Disclaimer

Anteon is created for testing the performance of web applications. Users must be the owner of the target system. Using it for harmful purposes is extremely forbidden. Anteon team & company is not responsible for its’ usages and consequences.

## 📜 License

Licensed under the [AGPLv3](LICENSE)
