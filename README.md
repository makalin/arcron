# Arcron

**AI-Powered Autonomous Cron Agent**  
A dynamic task scheduler that learns system patterns and optimizes job schedules based on predicted system load — moving beyond static time-based cron jobs.

---

## 🚀 Overview

Arcron is a next-generation replacement for traditional cron.  
Instead of executing jobs at rigid intervals, Arcron leverages **machine learning** and **system monitoring** to intelligently adapt scheduling in real time.

- Continuously monitors CPU, memory, I/O, and network utilization  
- Predicts optimal times for resource-intensive tasks  
- Self-corrects schedules through a continuous feedback loop  
- Written in **Go** for high performance and portability  

---

## ✨ Features

### Core Features
- **Dynamic Scheduling** – Optimizes execution based on predicted system load  
- **Machine Learning Core** – Learns patterns from historical system metrics  
- **Resource Awareness** – Avoids collisions with peak usage periods  
- **Self-Healing** – Adjusts schedules when predictions deviate from reality  
- **System Agnostic** – Works across Linux-based environments

### Pro Features ✨
- **🌐 Web Dashboard** – Beautiful, real-time web interface for monitoring and job management
- **🤖 Advanced ML Models** – Seasonality detection, anomaly detection, and LSTM-based predictions
- **📊 Prometheus Metrics** – Export metrics for monitoring and alerting
- **🔔 Multi-Channel Alerting** – Email, Slack, and webhook notifications
- **📡 Real-Time Updates** – WebSocket support for live metrics and job status
- **🔐 Authentication Ready** – Dashboard authentication support
- **📈 Advanced Analytics** – Job statistics, execution history, and performance insights
- **🛠️ RESTful API** – Complete API for integration and automation  

---

## 🛠️ Tech Stack

- **Language**: Go 1.22+
- **Machine Learning**: Custom ML models with seasonality & anomaly detection
- **Monitoring**: gopsutil for system metrics, real-time WebSocket updates
- **Persistence**: SQLite with GORM
- **Web Framework**: Gorilla Mux & WebSocket
- **Frontend**: Vanilla JavaScript with Chart.js
- **Metrics**: Prometheus-compatible exporter  

---

## 📦 Installation

```bash
git clone https://github.com/makalin/arcron.git
cd arcron
go mod download
go build -o arcron ./cmd/arcron
./arcron --help
```

Or use the Makefile:

```bash
make build
make run
```

---

## ⚙️ Usage

Define jobs in a YAML/JSON config:

```yaml
jobs:
  - name: backup
    command: "rsync -av /data /backup"
    type: "resource-intensive"
  - name: logrotate
    command: "logrotate /etc/logrotate.conf"
    type: "light"
```

Run Arcron:

```bash
./arcron --config config/arcron.yaml
```

Arcron will **predict execution windows** and optimize task runs accordingly.

### Web Dashboard

Start Arcron with the dashboard enabled (default):

```bash
./arcron --config config/arcron.yaml --dashboard
```

Then open your browser to `http://localhost:8080` to access the web dashboard.

### API Endpoints

- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/{name}` - Get job details
- `POST /api/v1/jobs/{name}/execute` - Execute a job manually
- `GET /api/v1/metrics` - Get system metrics
- `GET /api/v1/ml/status` - Get ML engine status
- `GET /api/v1/system/status` - Get system status
- `WS /ws` - WebSocket for real-time updates

### Prometheus Metrics

If enabled in config, metrics are available at `http://localhost:9090/metrics`

---

## 📊 Roadmap

* [x] Web dashboard for monitoring & overrides ✅
* [x] Advanced ML models (seasonality + anomaly detection) ✅
* [x] Prometheus metrics exporter ✅
* [x] Multi-channel alerting (Email, Slack, Webhooks) ✅
* [ ] Kubernetes integration
* [ ] Distributed scheduling across multiple nodes
* [ ] Job templates and presets
* [ ] Backup and restore functionality
* [ ] Advanced authentication (OAuth2, JWT)

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!
Feel free to fork the repo and submit a PR.

---

## 📜 License

MIT License © 2025

---

## 🔗 Badges

![Go](https://img.shields.io/badge/Go-1.22-blue)
![Build](https://img.shields.io/github/actions/workflow/status/makalin/arcron/go.yml)
![License](https://img.shields.io/badge/license-MIT-green)
![Status](https://img.shields.io/badge/status-experimental-orange)
