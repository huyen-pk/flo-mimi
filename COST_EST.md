> **WARNING: AI generated content, take it with a grain of salt.**

# Infrastructure Cost Estimation: Flo-Mimi Stack

This document outlines the infrastructure cost estimation required to host a multi-tenant ingestion, storage, and analytical query architecture (utilizing Redpanda, ClickHouse, Iceberg/MinIO, and Trino) for up to 1,000,000 SME clients.

## Baseline Architecture Assumptions (100,000 SMEs)
The stack operates as a **high-throughput, event-driven architecture**.
*   **"Small" SME Profile:** ~100 events/day/SME
*   **Total Expected Volume:** 10,000,000 events/day
*   **Peak Ingestion:** 500 events/sec

---

## 1. Multi-Tenant Infrastructure Cost (Monthly)

Scaling to 100k clients necessitates transitioning from single Virtual Machines to a Managed Kubernetes (e.g., EKS/GKE) environment for compute and orchestration.

| Component | Recommended Cloud Resource (AWS/GCP Equivalent) | Est. Monthly Cost |
| :--- | :--- | :--- |
| **Ingestion (Redpanda)** | 3x i4i.large (NVMe optimized instances) | **$800 - $1,200** |
| **Hot Serving (ClickHouse)** | 2x r6id.xlarge (Memory/SSD optimized) | **$600 - $900** |
| **Compute (Trino/App/Gateway)** | Managed Kubernetes Cluster (approx. 32-64 vCPU) | **$1,000 - $1,500** |
| **Storage (Object Storage)** | AWS S3 / GCS (5TB Hot + 50TB Cold) | **$1,300 - $1,800** |
| **Managed DB (Postgres)** | RDS Aurora (db.r6g.xlarge) + Multi-AZ | **$450 - $600** |
| **Observability (Managed)** | Managed Grafana/Prometheus/Loki (Grafana Cloud or similar) | **$500 - $1,000** |
| **Total Monthly Estimate** | | **$4,650 - $7,100** |

---

## 2. Strategic Cost Analysis (Cost Centers)

### A. The "Storage Paradox" (Events vs. Metrics)
At the 100,000 SME scale, **Observability** costs can easily rival **Data** infrastructure costs if not optimized.
*   **Events:** 10M events/day represents roughly 300GB/month of raw data. In highly-durable object storage (like AWS S3 or MinIO), raw storage is negligible (~$7/month).
*   **Logs & Traces:** Comprehensive, unfiltered logging for 100k clients generates terabytes of raw logs. Leveraging **Tempo/Loki with S3 backends** and strictly implementing tiered sampling policies is mandatory to prevent runaway billing and keep observability infrastructure under $1,000/month.

### B. The "Idle Compute" Cost
Lightweight, highly-available services (e.g., `event-gateway` and `platform`) often sit underutilized during off-peak hours.
*   **Scaling Tip:** Implement strict **Horizontal Pod Autoscaling (HPA)** limits. Systematically scaling compute down by ~70% during "off-hours" (nights/weekends) reliably limits unnecessary compute overhead and can translate to approximately **$800/month** in direct savings.

---

## 3. Scaling to 1,000,000 SMEs (The "Step Function")

Transitioning from 100k to 1M clients typically does not invoke a linear 10x cost increase. Driven by Kappa architecture efficiencies, the expectation aligns closer to a **4x - 6x** cost increase.

*   **Redpanda Tiered Storage:** Cluster sizing relies on extending cold storage capacity (cheap object storage) instead of perpetually adding expensive local disk.
*   **ClickHouse Sharding:** Growth necessitates a migration to a **Sharded Cluster** architecture or a managed **ClickHouse Cloud** offering.
*   **Estimated Cost for 1M SMEs:** **$18,000 - $25,000 / month.**

---

## 4. Immediate Cost Reduction Strategies (40% Savings Target)

1.  **Spot Instances:** Provision stateless workloads, specifically **Trino workers** and **Dagster executors**, onto Spot instances. Infrastructure preemption handles gracefully via orchestrator retries without risking persistent data loss.
2.  **Aggressive TTL Policies:** Limit retention of hot "Product Engagement" data in localized ClickHouse SSDs to an absolute maximum of 7 days. Immediately and systematically transition older data to object storage managed via the MinIO/S3 Iceberg catalog.
3.  **Graviton (ARM64) Infrastructure:** Modern Docker images (including Redpanda and ClickHouse) support native ARM processing. Migrating the orchestration footprint to AWS Graviton instances routinely yields an immediate **20% price/performance improvement**.

---

## 5. Granular Cost Breakdown & Billing Units

The monthly cost estimations rely on **Standard On-Demand Hourly Rates** for cloud resources in the **US-East-1 (N. Virginia)** region, forecasted for **April 2026**.

Each component evaluates specific infrastructure units relative to scaling requirements within a production deployment.

### A. Ingestion: Redpanda
*   **Primary Units:** **vCPU & IOPS.**
*   **Pricing Factor:** AWS **i4i.large** instance baseline.
*   **Logic:** At ~$0.172/hour (~$125/month) per node, a standard 3-node cluster for High Availability (HA) provisions at ~$375/month. Sustaining the peak IOPS and network bandwidth requirements of 1M clients necessitates utilizing higher-tier i4i instances or provisioning additional attached storage overhead, bridging the estimate to **$800+**.

### B. Hot Serving: ClickHouse
*   **Primary Units:** **RAM & Local NVMe.**
*   **Pricing Factor:** AWS **r6id.xlarge** instance.
*   **Logic:** ClickHouse leverages in-memory data parts to sustain analytic query speed. An `r6id.xlarge` (4 vCPU, 32GB RAM) incurs ~$220/month. Supporting 100k SMEs mandates a minimum 2-node cluster to prevent downtime during maintenance operations, plus the inclusion of high-performance EBS/NVMe storage volumes.

### C. Data Lake: S3 / MinIO
*   **Primary Units:** **GB-Months & API Requests.**
*   **Pricing Factor:** AWS S3 Standard Tier.
*   **Logic:** 
    *   **Storage:** ~$0.023 per GB. 
    *   **Egress:** ~$0.09 per GB (a critical metric that often drives unbudgeted cost overruns).
    *   **Requests:** ~$0.005 per 1,000 PUT requests. For 10M events per day natively ingested, request costs can quickly eclipse raw storage costs if data is not correctly micro-batched prior to upload.

### D. Relational Metadata: Postgres (AppDB)
*   **Primary Units:** **Instance Class (Managed).**
*   **Pricing Factor:** AWS **Aurora RDS db.r6g.xlarge**.
*   **Logic:** A managed `db.r6g.xlarge` instances incurs ~$328/month. As this database serves as the core SME operational source of truth, a **Multi-AZ** configuration (2x native cost factor) is imperative for disaster recovery resilience, bringing the estimate to ~$650.

### E. Orchestration: Kubernetes (EKS/GKE)
*   **Primary Units:** **Cluster Management Fee + Worker Nodes.**
*   **Pricing Factor:** **Flat hourly Control Plane fee (~$0.10/hour).**
*   **Logic:** Managed Kubernetes services incur a flat ~$73/month baseline to sustain the Control Plane. The aggregate orchestration cost evaluates the sum of the underlying EC2 instances executing workloads like `event-gateway`, `platform`, and `dagster`.

---

### Summary of Scaling Units

| Component | Primary Scaling Unit | Secondary Scaling Unit |
| :--- | :--- | :--- |
| **Compute** | Hourly Instance Rate | Data Transfer (Egress) |
| **Storage** | GB-Month | IOPS / Throughput |
| **Database** | Instance Size | Backup Storage / IOPS |
| **Streaming** | vCPU Count | Local NVMe Capacity |

**Strategic Provisioning Note:** These unit costs shift dramatically when utilizing **Spot Instances** (yielding up to 70% cost reduction for stateless workloads) or **Reserved Instances / Savings Plans** (yielding up to 40% cost reduction for persistent stateful clusters). The provided estimates intentionally utilize standard "On-Demand" pricing to construct a conservative, worst-case budget model.

---

## 6. Summary Verdict

At a baseline scale of **100,000 SMEs**, the current proposed stack is remarkably efficient, yielding roughly **$0.05 - $0.07 per SME per month** in infrastructure costs. This baseline provides substantial margin flexibility, remaining highly profitable even if the commercial service is provided at a low entry price point (e.g., $1-$5/month).
