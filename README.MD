# EKS Fargate Private Flask Challenge — Revised

> Private EKS, Fargate, IRSA, ALB, DynamoDB, Secrets Manager, CloudWatch, HPA를 통합해서 구성하는 실전형 AWS/EKS 모의고사입니다.

---

## 정보

모든 주요 리소스에는 다음 태그를 적용합니다.

```text
Project=EKSChallenge
Owner=<본인 이름>
```

---

# 전체 요구 구조

## 서비스 트래픽

```text
Internet
   ↓
Internet-facing ALB
   ↓
Gateway API / HTTPRoute
   ↓
Flask Pods on Fargate
   ├── DynamoDB
   ├── Secrets Manager
   └── CloudWatch Logs
```

## 관리자 접근

```text
Administrator
   ↓
SSM
   ↓
Private Management EC2
   ↓
Private-only EKS API Endpoint
```

## EKS Compute 구조

```text
Managed Node Group
├── AWS Load Balancer Controller
├── Metrics Server
└── 시스템 관리용 워크로드

Fargate
└── Flask Application Pods
```

---

# 1. VPC — 15점

## 기본 설정

| 항목 | 값 |
|---|---|
| VPC Name | `skills-eks-vpc` |
| VPC CIDR | `10.80.0.0/16` |
| DNS Support | 활성화 |
| DNS Hostnames | 활성화 |

## Subnet

| 이름 | AZ | CIDR | 용도 |
|---|---|---|---|
| `skills-public-a` | 2a | `10.80.1.0/24` | ALB, NAT Gateway |
| `skills-public-c` | 2c | `10.80.2.0/24` | ALB |
| `skills-eks-a` | 2a | `10.80.11.0/24` | Node, Fargate |
| `skills-eks-c` | 2c | `10.80.12.0/24` | Node, Fargate |
| `skills-admin-a` | 2a | `10.80.21.0/24` | Management EC2 |

## 조건

- Public Subnet만 Internet Gateway 경로를 가집니다.
- EKS 및 Admin Subnet에는 Public IPv4를 할당하지 않습니다.
- `skills-public-a`에 NAT Gateway 1개를 생성합니다.
- EKS/Admin Subnet은 필요 시 NAT Gateway를 통해 외부 AWS API에 접근할 수 있어야 합니다.
- Public Subnet에는 internet-facing ALB 자동 검색이 가능하도록 태그를 설정합니다.
- EKS Subnet에는 내부 Load Balancer 검색이 가능하도록 태그를 설정합니다.

## 필수 VPC Endpoint

### Gateway Endpoint

- S3
- DynamoDB

### Interface Endpoint

- ECR API
- ECR DKR
- Secrets Manager
- CloudWatch Logs

> 모든 Interface Endpoint에는 **Private DNS**를 활성화합니다.

## 최소 동작 조건

다음 기능이 정상 동작해야 합니다.

- ECR 인증 및 이미지 다운로드
- ECR 이미지 Layer 다운로드
- DynamoDB 접근
- Secrets Manager 접근
- CloudWatch Logs 전송

---

# 2. EKS 및 관리 환경 — 20점

## EKS Cluster

| 항목 | 값 |
|---|---|
| Cluster Name | `skills-private-cluster` |
| IP Family | IPv4 |
| Public Endpoint | 비활성화 |
| Private Endpoint | 활성화 |
| Authentication Mode | `API` 또는 `API_AND_CONFIG_MAP` |

## 조건

- Kubernetes API Endpoint는 인터넷에서 접근할 수 없어야 합니다.
- Control Plane Logs를 활성화합니다.

최소 다음 로그를 활성화합니다.

```text
api
audit
authenticator
controllerManager
scheduler
```

## Managed Node Group

| 항목 | 값 |
|---|---|
| Name | `system-nodegroup` |
| Instance Type | `t3.medium` |
| Minimum | 2 |
| Desired | 2 |
| Maximum | 3 |
| Subnet | `skills-eks-a`, `skills-eks-c` |

## 조건

- EC2 Node에는 Public IPv4를 할당하지 않습니다.
- 두 AZ에 분산합니다.
- AWS Load Balancer Controller는 Managed Node Group에서 실행합니다.
- Metrics Server는 Managed Node Group에서 실행합니다.
- Flask Application은 Managed Node Group에서 실행하면 안 됩니다.

## Management EC2

| 항목 | 값 |
|---|---|
| Name | `skills-eks-admin` |
| Subnet | `skills-admin-a` |
| Instance Type | `t3.micro` |
| Connection | SSM Session Manager |

### 설치 도구

```text
AWS CLI v2
kubectl
Helm
jq
curl
```

### 정상 동작 확인

```bash
aws sts get-caller-identity

aws eks update-kubeconfig \
  --name skills-private-cluster

kubectl get nodes

kubectl get pods -A
```

### 금지

- SSH 사용 금지
- Public IP 사용 금지

---

# 3. ECR 및 Flask 이미지 — 10점

## ECR Repository

| 항목 | 값 |
|---|---|
| Repository Name | `skills-flask` |
| Tag Mutability | Immutable |
| Scan on Push | 활성화 |
| Encryption | KMS |

Flask Application을 직접 작성하고 Container Image로 빌드합니다.

## Container 조건

- Root 사용자로 실행하지 않습니다.
- Container Port는 `8080`을 사용합니다.
- `latest` Tag를 사용하지 않습니다.
- `/health`는 외부 서비스 상태와 무관하게 `200`을 반환해야 합니다.
- 로그는 `stdout` / `stderr`로 출력합니다.
- AWS Access Key를 이미지 또는 환경변수에 저장하면 안 됩니다.

## 필수 API

### `GET /health`

응답:

```json
{
  "status": "healthy"
}
```

### `GET /items`

DynamoDB의 Item 목록을 반환합니다.

### `POST /items`

Request:

```json
{
  "name": "keyboard",
  "price": 10000
}
```

DynamoDB 저장 후 응답:

```json
{
  "id": "<생성된 ID>",
  "name": "keyboard",
  "price": 10000
}
```

### `GET /config`

Secrets Manager의 값을 조회하되 실제 Secret 값 전체를 노출하면 안 됩니다.

응답:

```json
{
  "secret_loaded": true
}
```

---

# 4. Fargate 배포 — 15점

## Namespace

```text
skills
```

## Fargate Profile

| 항목 | 값 |
|---|---|
| Profile Name | `flask-fargate-profile` |
| Selector Namespace | `skills` |

## Kubernetes Resource

```text
ServiceAccount: flask-sa
Deployment:     flask-app
Service:        flask-service
ConfigMap:      flask-config
```

## Deployment 조건

- Replica: `2`
- Flask Pod는 반드시 Fargate에서 실행
- Container Port: `8080`
- Rolling Update 사용
- Readiness Probe: `/health`
- Liveness Probe: `/health`
- CPU Request / Limit 설정
- Memory Request / Limit 설정
- 서로 다른 AZ에 최대한 분산
- Service Type: `ClusterIP`
- `terminationGracePeriodSeconds` 설정

## Fargate 확인

다음 정보로 Fargate 실행 여부를 판별할 수 있어야 합니다.

```text
eks.amazonaws.com/compute-type=fargate
```

---

# 5. IAM 및 IRSA — 15점

다음 IAM Role을 분리합니다.

```text
EKS Cluster Role
Managed Node Group Role
Fargate Pod Execution Role
Flask Application Role
Load Balancer Controller Role
Management EC2 Role
```

## Fargate Pod Execution Role

다음 용도로만 사용합니다.

- ECR 이미지 다운로드
- Fargate Pod 실행

## Flask Application Role

다음 권한만 부여합니다.

- 지정 DynamoDB Table 접근
- 지정 Secrets Manager Secret 조회
- 필요한 KMS 복호화

## 조건

- Flask Pod에 Access Key 저장 금지
- `flask-sa`가 Flask Application Role 사용
- Application Role과 Pod Execution Role 혼용 금지
- DynamoDB Resource에 불필요한 `*` 사용 금지
- Secrets Manager Resource에 불필요한 `*` 사용 금지
- Fargate Flask 권한에는 **IRSA** 사용

## IRSA Trust Policy 제한

```text
Namespace:      skills
ServiceAccount: flask-sa
```

AWS Load Balancer Controller는 다음을 별도로 사용합니다.

```text
Dedicated ServiceAccount
Dedicated IAM Role
IRSA
```

Flask Application Role과 Load Balancer Controller Role은 반드시 분리합니다.

---

# 6. DynamoDB 및 Secrets Manager — 10점

## DynamoDB

| 항목 | 값 |
|---|---|
| Table Name | `skills-items` |
| Partition Key | `id` |
| Key Type | String |
| Billing Mode | `PAY_PER_REQUEST` |
| Encryption | KMS |
| Point-in-Time Recovery | 활성화 |

Flask Application Role에는 다음 작업만 허용합니다.

```text
dynamodb:PutItem
dynamodb:GetItem
dynamodb:Scan
```

## Secrets Manager

| 항목 | 값 |
|---|---|
| Secret Name | `skills/flask/config` |
| Encryption | KMS |

예시 Secret:

```json
{
  "environment": "competition",
  "message": "private-value"
}
```

## 조건

- Flask는 실행 시점에 Secrets Manager에서 값을 조회해야 합니다.
- Secret 값을 Kubernetes Secret에 복사하면 안 됩니다.
- Secret 값을 Container Image에 포함하면 안 됩니다.

---

# 7. ALB + Gateway API — 10점

AWS Load Balancer Controller를 Helm으로 설치합니다.

## 조건

- Controller는 `system-nodegroup`에서 실행
- 전용 ServiceAccount 사용
- 전용 IRSA Role 사용
- Flask Application Role과 분리
- internet-facing ALB 생성
- ALB는 Public Subnet A/C 사용
- Target Type: `ip`

## Gateway API Resource

다음 Resource를 생성합니다.

```text
Gateway
HTTPRoute
```

## HTTPRoute

다음 경로를 `flask-service`로 전달합니다.

```text
/health
/items
/config
```

## 검증 조건

- 등록된 Target 두 개가 모두 `Healthy`
- 정의되지 않은 경로는 `404` 또는 지정된 고정 응답 반환

---

# 8. CloudWatch Logging 및 HPA — 5점

## CloudWatch Log Group

```text
/skills/eks/flask
```

Retention:

```text
30 days
```

## 조건

- Flask stdout 로그가 CloudWatch에서 확인되어야 합니다.
- `POST /items` 요청의 Method와 Path를 기록합니다.
- Secret 실제 값은 로그에 기록하면 안 됩니다.

## Horizontal Pod Autoscaler

| 항목 | 값 |
|---|---|
| Target | `flask-app` |
| Minimum Pods | 2 |
| Maximum Pods | 6 |
| CPU Target | 60% |

Metrics Server는 `system-nodegroup`에서 실행합니다.

부하 종료 후 Flask Pod 수가 다시 `2`로 감소해야 합니다.

---

# 금지 사항

- EKS Public Endpoint 활성화
- Flask Pod에 Access Key 저장
- Flask Pod가 Managed Node Group에서 실행
- Application Role에 `AdministratorAccess` 연결
- 모든 포트를 `0.0.0.0/0`에 허용
- `latest` 이미지 사용
- DynamoDB Resource에 불필요한 `*`
- Secrets Manager Resource에 불필요한 `*`
- Management EC2에 Public IP 부여
- Management EC2에 SSH 개방

---

# 채점표

| 영역 | 배점 |
|---|---:|
| VPC · Subnet · Endpoint | 15 |
| Private EKS · 관리 환경 | 20 |
| ECR · Flask 이미지 | 10 |
| Fargate 배포 | 15 |
| IAM · IRSA | 15 |
| DynamoDB · Secrets Manager | 10 |
| ALB · Gateway API | 10 |
| CloudWatch · HPA | 5 |
| **합계** | **100** |

---

# 권장 진행 순서

처음 도전할 때는 다음 순서로 진행하는 것을 권장합니다.

```text
VPC / Subnet
   ↓
Private EKS
   ↓
Managed Node Group
   ↓
Management EC2 + SSM
   ↓
ECR
   ↓
Fargate
   ↓
IRSA
   ↓
DynamoDB / Secrets Manager
   ↓
Gateway API
   ↓
AWS Load Balancer Controller
   ↓
ALB Healthy
   ↓
CloudWatch
   ↓
HPA
```

핵심 목표는 다음 흐름을 완성하는 것입니다.

```text
Private EKS
→ Fargate 실행
→ IRSA
→ ALB Healthy
→ DynamoDB 저장
→ Secrets Manager 조회
→ HPA 동작
```
