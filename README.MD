# EKS Fargate Private Gin REST API Challenge — Revised

## 목표 시간
첫 시도: 5시간
숙련 목표: 3시간 30분
만점: 100점
보너스: 15점
## 리전
ap-northeast-2
모든 주요 리소스에는 다음 태그를 적용합니다.
Project=EKSChallenge
Owner=<본인 이름>

---

# 전체 요구 구조

## 서비스 트래픽
Internet
↓
Internet-facing ALB
↓
Gateway API / HTTPRoute
↓
Gin REST API Pods on Fargate
├─ DynamoDB
├─ Secrets Manager
└─ CloudWatch Logs
## 관리자 접근
Administrator
↓
SSM
↓
Private Management EC2
↓
Private-only EKS API Endpoint
## 클러스터 Compute 구조
## Managed Node Group
├─ AWS Load Balancer Controller
├─ Metrics Server
└─ 시스템 관리용 워크로드
Fargate
└─ Gin REST API Application Pods

---

# 제1항목: VPC — 15점

다음 VPC를 구성합니다.
VPC Name: skills-eks-vpc
VPC CIDR: 10.80.0.0/16
DNS Support: 활성화
DNS Hostnames: 활성화
## 서브넷 구성

| **이름**          | **AZ** | **CIDR**      | **용도**           |
| --------------- | ------ | ------------- | ---------------- |
| skills-public-a | 2a | 10.80.1.0/24  | ALB, NAT Gateway |
| skills-public-c | 2c | 10.80.2.0/24  | ALB              |
| skills-eks-a    | 2a | 10.80.11.0/24 | Node, Fargate    |
| skills-eks-c    | 2c | 10.80.12.0/24 | Node, Fargate    |
| skills-admin-a  | 2a | 10.80.21.0/24 | Management EC2   |

## 조건

- Public Subnet만 Internet Gateway 경로를 가집니다.
- EKS와 Admin Subnet에는 Public IPv4를 할당하지 않습니다.
- skills-public-a에 NAT Gateway 1개를 생성합니다.
- EKS/Admin Subnet은 필요 시 NAT Gateway를 통해 외부 AWS API에 접근할 수 있어야 합니다.
- Public Subnet은 internet-facing ALB 자동 검색이 가능하도록 태그를 설정합니다.
- EKS Subnet은 내부 Load Balancer 검색이 가능하도록 태그를 설정합니다.

## 필수 VPC Endpoint
### Gateway Endpoint

- S3
- DynamoDB

### Interface Endpoint

- ECR API
- ECR DKR
- Secrets Manager
- CloudWatch Logs

모든 Interface Endpoint에는 Private DNS를 활성화합니다.
### 최소 동작 조건

- ECR 인증 및 이미지 다운로드
- ECR Image Layer 다운로드
- DynamoDB 접근
- Secrets Manager 접근
- CloudWatch Logs 전송

---

# 제2항목: EKS와 관리 환경 — 20점

## EKS Cluster
Cluster Name: skills-private-cluster
IP Family: IPv4
Public Endpoint: 비활성화
Private Endpoint: 활성화
Authentication Mode: API 또는 API\_AND\_CONFIG\_MAP
## 조건

- Kubernetes API Endpoint는 인터넷에서 접근할 수 없어야 합니다.
- Control Plane Logs를 활성화합니다.

다음 로그를 모두 활성화합니다.
api
audit
authenticator
controllerManager
scheduler
## Managed Node Group
Name: system-nodegroup
Instance Type: t3.medium
Minimum: 2
Desired: 2
Maximum: 3
## Subnet
skills-eks-a
skills-eks-c
## 조건

- EC2 Node에 Public IPv4를 할당하지 않습니다.
- 두 AZ에 최대한 분산합니다.
- AWS Load Balancer Controller는 Node Group에서 실행합니다.
- Metrics Server는 Node Group에서 실행합니다.
- Gin REST API Application은 Node Group에서 실행하면 안 됩니다.

## Management EC2
Name: skills-eks-admin
Subnet: skills-admin-a
Instance Type: t3.micro
Connection: SSM Session Manager
### 설치 도구
AWS CLI v2
kubectl
Helm
jq
curl
다음 명령이 정상 동작해야 합니다.
aws sts get-caller-identity
aws eks update-kubeconfig --name skills-private-cluster --region ap-northeast-2
kubectl get nodes
kubectl get pods -A
## 금지

- SSH 사용 금지
- Public IP 사용 금지

---

# 제3항목: ECR과 Gin 이미지 — 10점

## ECR Repository
Repository Name: skills-gin
Tag Mutability: Immutable
Scan on Push: 활성화
Encryption: KMS
Go와 Gin을 사용하여 REST API Application을 직접 작성하고 Container Image로 빌드합니다.
### Application 조건

- Go 사용
- Gin Framework 사용
- AWS SDK for Go v2 사용
- HTTP Server Port: 8080
- Graceful Shutdown 구현
- SIGTERM 처리
- AWS Access Key를 코드에 직접 작성하면 안 됨

### Container 조건

- Root 사용자 실행 금지
- Container Port: 8080
- latest Tag 사용 금지
- Multi-stage Build 사용
- Go Binary를 Build Stage에서 컴파일
- Runtime Image에는 Go Compiler가 포함되지 않아도 됨
- /health는 외부 서비스 상태와 관계없이 200 반환
- 로그는 stdout/stderr 사용
- AWS Access Key를 이미지 또는 환경변수에 저장 금지

### 권장 Build 구조
Go Source
↓
Build Stage
↓
Compiled Go Binary
↓
Minimal Runtime Image
↓
Non-root User
### 필수 REST API
GET /health
### 응답
{
"status": "healthy"
}
## 조건

- DynamoDB 상태와 무관하게 정상 동작
- Secrets Manager 상태와 무관하게 정상 동작
- HTTP Status Code 200

GET /items
DynamoDB의 Item 목록을 조회합니다.
### 사용 작업
dynamodb\:Scan
### 예시 응답
[
{
"id": "1234",
"name": "keyboard",
"price": 10000
}
]
POST /items
### Request
{
"name": "keyboard",
"price": 10000
}
Application은 새로운 ID를 생성한 후 DynamoDB에 저장합니다.
### 사용 작업
dynamodb\:PutItem
### 응답 예시
{
"id": "<생성된 ID>",
"name": "keyboard",
"price": 10000
}
### 성공 시 HTTP Status Code
201 Created
GET /config
실행 시점에 Secrets Manager의 다음 Secret을 조회합니다.
skills/gin/config
### 응답
{
"secret\_loaded": true
}
## 조건

- 실제 Secret 값 전체를 응답에 포함하면 안 됩니다.
- 실제 Secret 값을 로그에 기록하면 안 됩니다.
- Secret 조회 실패 시 적절한 오류 응답을 반환해야 합니다.

---

# 제4항목: Fargate 배포 — 15점

## Namespace
skills
## Fargate Profile
Profile Name: gin-fargate-profile
Selector Namespace: skills
## Kubernetes Resource
ServiceAccount: gin-sa
Deployment: gin-app
Service: gin-service
ConfigMap: gin-config
### Deployment 조건

- Replica: 2
- Gin Pod는 반드시 Fargate에서 실행
- Container Port: 8080
- Rolling Update 사용
- Readiness Probe: /health
- Liveness Probe: /health
- CPU Request / Limit 설정
- Memory Request / Limit 설정
- 서로 다른 AZ에 최대한 분산
- Service Type: ClusterIP
- terminationGracePeriodSeconds 설정

Application은 SIGTERM을 받아 정상적으로 종료될 수 있어야 합니다.
Kubernetes Pod 종료
↓
SIGTERM
↓
Gin HTTP Server Shutdown
↓
기존 Request 처리 완료
↓
Process 종료
다음 Label 또는 정보로 Fargate 실행 여부를 확인할 수 있어야 합니다.
eks.amazonaws.com/compute-type=fargate

---

# 제5항목: IAM과 IRSA — 15점

다음 IAM Role을 분리합니다.
EKS Cluster Role
Managed Node Group Role
Fargate Pod Execution Role
### Gin Application Role
Load Balancer Controller Role
Management EC2 Role
Fargate Pod Execution Role
### 용도

- ECR 이미지 다운로드
- Fargate Pod 실행에 필요한 권한

Gin Application의 DynamoDB 또는 Secrets Manager 접근용으로 사용하면 안 됩니다.
### Gin Application Role
다음 권한을 가집니다.

- 지정 DynamoDB Table 접근
- 지정 Secrets Manager Secret 조회
- 필요한 KMS 복호화

## 조건

- Gin Pod에 Access Key 저장 금지
- gin-sa가 Gin Application Role 사용
- Application Role과 Pod Execution Role 혼용 금지
- DynamoDB Resource에 불필요한 \* 금지
- Secrets Manager Resource에 불필요한 \* 금지
- Fargate Gin Application 권한은 IRSA 사용

### IRSA Trust Policy 제한
Namespace: skills
ServiceAccount: gin-sa
Go Application은 AWS SDK for Go v2의 Default Credential Provider Chain을 사용해야 합니다.
### 예시
config.LoadDefaultConfig(...)
Application 코드 내부에서 다음 값을 직접 설정하면 안 됩니다.
AWS\_ACCESS\_KEY\_ID
AWS\_SECRET\_ACCESS\_KEY
AWS\_SESSION\_TOKEN
## AWS Load Balancer Controller
AWS Load Balancer Controller는 별도의 ServiceAccount와 별도의 IAM Role을 사용합니다.
AWS Load Balancer Controller Pod
↓
Dedicated ServiceAccount
↓
IRSA
↓
Load Balancer Controller IAM Role
Gin Application Role과 반드시 분리합니다.

---

# 제6항목: DynamoDB · Secrets Manager — 10점

## DynamoDB
Table Name: skills-items
Partition Key: id
Partition Key Type: String
Billing Mode: PAY\_PER\_REQUEST
Encryption: KMS
Point-in-Time Recovery: 활성화
Gin Application Role에는 다음 작업만 허용합니다.
dynamodb\:PutItem
dynamodb\:GetItem
dynamodb\:Scan
DynamoDB Resource에는 해당 Table ARN을 지정합니다.
불필요한 다음 설정은 금지합니다.
"Resource": "\*"
## Secrets Manager
Secret Name: skills/gin/config
Encryption: KMS
### 예시 Secret
{
"environment": "competition",
"message": "private-value"
}
Gin Application은 API 요청 처리 시 Secrets Manager에서 값을 조회해야 합니다.
다음 방식은 인정하지 않습니다.

- Secret 값을 Container Image에 저장
- Secret 값을 ConfigMap에 저장
- Secret 값을 Kubernetes Secret에 복사
- Secret 값을 환경변수에 직접 저장

### 허용 구조
Gin Pod
↓
IRSA
↓
Secrets Manager Interface Endpoint
↓
## Secrets Manager

---

# 제7항목: ALB + Gateway API — 10점

AWS Load Balancer Controller를 Helm으로 설치합니다.
## 조건

- Controller는 system-nodegroup에서 실행
- 전용 ServiceAccount 사용
- 전용 IRSA Role 사용
- Gin Application Role과 분리
- internet-facing ALB 생성
- ALB는 Public Subnet A/C 사용
- Target Type: ip

## Gateway API Resource
Gateway
HTTPRoute
Gateway는 외부 HTTP Traffic을 받을 Listener를 정의합니다.
### 예시
Protocol: HTTP
Port: 80
HTTPRoute는 다음 경로를 gin-service로 전달합니다.
/health
/items
/config
### 논리 구조
Gateway
↓
HTTPRoute
↓
gin-service
↓
Gin Pod Endpoints
AWS Load Balancer Controller는 해당 구성을 감시하고 실제 AWS ALB 구성을 생성하거나 수정합니다.
### 실제 Data Plane
Internet
↓
ALB
↓
Listener
↓
Listener Rule
↓
Target Group
↓
Gin Fargate Pod IP :8080
### 검증 조건

- internet-facing ALB가 생성되어야 함
- Target Type이 ip여야 함
- Gin Fargate Pod IP가 Target으로 등록되어야 함
- 등록된 Target 두 개가 모두 Healthy
- /health 정상 접근
- /items 정상 접근
- /config 정상 접근
- 정의되지 않은 경로는 404 또는 지정된 고정 응답 반환

---

# 제8항목: 로깅과 오토스케일링 — 5점

## CloudWatch Log Group
/skills/eks/gin
## Retention
30일
### Application Logging 조건
Gin Application 로그는 stdout/stderr를 사용합니다.
최소 다음 정보가 CloudWatch Logs에서 확인되어야 합니다.
HTTP Method
Request Path
HTTP Status Code
### 예시
method=GET path=/health status=200
method=POST path=/items status=201
method=GET path=/config status=200
특히 POST /items 요청의 다음 정보를 확인할 수 있어야 합니다.
Method: POST
Path: /items
다음 값은 로그에 기록하면 안 됩니다.

- Secret 실제 값
- AWS Access Key
- AWS Secret Access Key
- AWS Session Token

Fargate stdout/stderr 로그를 CloudWatch Logs로 전달할 수 있도록 Logging 설정을 구성합니다.
## Horizontal Pod Autoscaler
Target: gin-app
Minimum Pods: 2
Maximum Pods: 6
CPU Target: 60%
Metrics Server는 system-nodegroup에서 실행합니다.
### 검증 조건
### 부하 발생 시
Gin Pods 2
↓
CPU 증가
↓
HPA Scale Out
↓
최대 6 Pods
### 부하 종료 후
CPU 감소
↓
HPA Scale In
↓
Gin Pods = 2
최종적으로 Gin Pod가 최소 Replica인 2개로 돌아와야 합니다.

---

# 금지 사항

다음 항목 발견 시 해당 영역 감점 또는 0점 처리

- EKS Public Endpoint 활성화
- Gin Pod에 AWS Access Key 저장
- Gin Pod가 Managed Node Group에서 실행
- Application Role에 AdministratorAccess
- 모든 포트를 0.0.0.0/0에 허용
- latest 이미지 사용
- Root 사용자로 Container 실행
- DynamoDB Resource에 불필요한 \*
- Secrets Manager Resource에 불필요한 \*
- Secret 값을 Container Image에 저장
- Secret 값을 ConfigMap에 저장
- Management EC2에 Public IP 할당
- Management EC2에 SSH 개방
- Fargate Pod Execution Role을 Gin Application 권한으로 사용

---

# 채점표

| **영역**                     | **배점** |
| -------------------------- | ------ |
| VPC · Subnet · Endpoint    | 15  |
| Private EKS · 관리 환경        | 20  |
| ECR · Gin 이미지              | 10  |
| Fargate 배포                 | 15  |
| IAM · IRSA                 | 15  |
| DynamoDB · Secrets Manager | 10  |
| ALB · Gateway API          | 10  |
| CloudWatch · HPA           | 5   |
| 합계                         | 100 |

---

# 보너스 — 15점

| **항목**                               | **점수** |
| ------------------------------------ | ------ |
| WAF Rate-based Rule을 ALB에 연결         | +5  |
| /admin/\* WAF 차단                     | +3  |
| NetworkPolicy 적용                     | +3  |
| Terraform으로 VPC · EKS · Endpoint 재생성 | +4  |
| 합계                                   | +15 |

---

# 최종 목표

## 서비스 흐름
Internet
↓
ALB
↓
Gateway API / HTTPRoute
↓
Gin Service
↓
Gin Pods on Fargate
├─ DynamoDB
├─ Secrets Manager
└─ CloudWatch Logs
## 권한 흐름
Gin Pod
↓
gin-sa
↓
IRSA
↓
Gin Application IAM Role
├─ DynamoDB
├─ Secrets Manager
└─ KMS
## 관리 흐름
Administrator
↓
SSM Session Manager
↓
Private Management EC2
↓
Private EKS API Endpoint
↓
kubectl / Helm
## Controller 흐름
Gateway / HTTPRoute / Service
↓
## AWS Load Balancer Controller
↓
AWS ELB API
↓
ALB / Listener / Rule / Target Group
## 최종 검증
Private EKS
→ Managed Node Group 정상
→ Gin Pods Fargate 실행
→ IRSA 정상
→ DynamoDB 저장/조회
→ Secrets Manager 조회
→ ALB Target Healthy
→ Gateway API Routing
→ CloudWatch Logging
→ HPA Scale Out / Scale In
