flowchart TD
    subgraph Device["ESP32 Sensor Device"]
        S1[Temp / Humidity Sensor]
        S2[ESP32 Firmware]
        S3[Wi-Fi Network<br/> Any ISP / NAT]
        
        S1 --> S2
        S2 -->|HTTPS POST<br/>Readings + Heartbeat| S3
    end

    subgraph Internet["Public Internet"]
        I1[Routing / NAT / TLS]
    end

    subgraph Backend["Go Backend Server"]
        B1[HTTP API Server]
        B2[Auth Middleware<br/> API Key / HMAC]
        B3[Ingestion Handler]
        B4[Device State Manager]
    end

    subgraph Storage["Persistence Layer"]
        D1[(Time-Series DB<br/>Readings)]
        D2[(Relational DB<br/>Devices)]
    end

    S3 --> I1 --> B1
    B1 --> B2 --> B3
    B3 --> D1
    B3 --> B4 --> D2


