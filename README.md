# OPC UA Time Series Hub

## Overview
The OPC UA Time Series Hub is a personal project designed to integrate various technologies into a cohesive
system for monitoring and storing time series data from OPC UA servers. This project uses InfluxDB2 and Telegraf
for data storage and collection, with a focus on the OPC UA protocol for industrial automation and IoT applications.
The main feature of this project is the `opc-ua-time-series-hub-web`, a web interface that allows users to browse
the OPC UA hierarchy and select tags for storage in InfluxDB.

### Components
- **InfluxDB V2**: Time series database used for storing the collected data.
- **Telegraf**: An agent for collecting, processing, aggregating, and writing metrics.
- **OPC UA Server (mcr.microsoft.com/iotedge/opc-plc)**: Simulates an OPC UA server generating random data.
- **MySQL Database**: Stores user settings and configurations.
- **OPC UA Time Series Hub API**: Custom service that interfaces with the OPC UA server and Telegraf.
- **OPC UA Time Series Hub Web**: The web interface for interacting with the OPC UA server's data hierarchy.

## Getting Started

To get started with the OPC UA Time Series Hub, clone this repository to your local machine and ensure Docker and
Docker Compose are installed.

```bash
git clone https://github.com/brentoncollins/opc-ua-time-series-hub.git
cd opc-ua-time-series-hub/deploy
docker-compose up -d
``` 

1. Navigate to `http://localhost:3000` in your web browser to access the web interface, add OPC UA Tags by expanding
   the OPC UA Hierarchy and selecting the checkboxes, click `Update Telegraf Config`

   ####  Note: If you see `Failed to fetch nodes. Please try again later.` The OPC Server is still starting, give it a minute and refresh the page. 

2. Navigate to `http://localhost:8086` in your web browser to access the InfluxDB2 interface with
   default User: root Password: password, which can be changed in the docker compose file. 

Have a look through the docker compose file to change the configuration for the OPC Server, 
see the documentation at mcr.microsoft.com/iotedge/opc-plc to see the available configuration options.
