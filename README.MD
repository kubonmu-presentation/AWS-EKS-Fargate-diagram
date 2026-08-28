EKS Fargate Private Flask Challenge — Revised
목표 시간:
첫 시도: 5시간
숙련 목표: 3시간 30분
만점: 100점
보너스: 15점
리전:
ap-northeast-2
모든 주요 리소스에 다음 태그를 적용합니다.
Project=EKSChallenge
Owner=<본인 이름>
전체 요구 구조
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

관리자
  ↓
SSM
  ↓
Private Management EC2
  ↓
Private-only EKS API Endpoint
클러스터에는 두 종류의 Compute를 사용합니다.
Managed Node Group
├── AWS Load Balancer Controller
├── Metrics Server
└── 시스템 관리용 워크로드

Fargate
└── Flask 애플리케이션 Pod

제1항목: VPC — 15점
다음 VPC를 구성합니다.
VPC Name: skills-eks-vpc
VPC CIDR: 10.80.0.0/16
DNS Support: 활성화
DNS Hostnames: 활성화
이름
AZ
CIDR
용도
skills-public-a
2a
10.80.1.0/24
ALB, NAT Gateway
skills-public-c
2c
10.80.2.0/24
ALB
skills-eks-a
2a
10.80.11.0/24
Node·Fargate
skills-eks-c
2c
10.80.12.0/24
Node·Fargate
skills-admin-a
2a
10.80.21.0/24
Management EC2

조건:
Public Subnet만 Internet Gateway 경로를 가짐
EKS와 Admin Subnet에는 Public IPv4를 할당하지 않음
skills-public-a에 NAT Gateway 1개 생성
EKS/Admin Subnet은 필요 시 NAT Gateway를 통해 외부 AWS API에 접근 가능
Public Subnet은 internet-facing ALB 자동 검색이 가능하도록 태그 설정
EKS Subnet은 내부 Load Balancer 검색이 가능하도록 태그 설정
다음 VPC Endpoint는 반드시 구성합니다.
Gateway Endpoint
- S3
- DynamoDB

Interface Endpoint
- ECR API
- ECR DKR
- Secrets Manager
- CloudWatch Logs
모든 Interface Endpoint에는 Private DNS를 활성화합니다.
최소 다음 기능이 동작해야 합니다.
ECR 인증 및 이미지 다운로드
ECR 이미지 Layer 다운로드
DynamoDB 접근
Secrets Manager 접근
CloudWatch Logs 전송

제2항목: EKS와 관리 환경 — 20점
EKS Cluster:
Cluster Name: skills-private-cluster
IP Family: IPv4
Public Endpoint: 비활성화
Private Endpoint: 활성화
Authentication Mode: API 또는 API_AND_CONFIG_MAP
조건:
Kubernetes API Endpoint는 인터넷에서 접근할 수 없어야 함
Control Plane Logs 활성화
최소 다음 로그 활성화
api
audit
authenticator
controllerManager
scheduler
Managed Node Group:
Name: system-nodegroup
Instance Type: t3.medium
Minimum: 2
Desired: 2
Maximum: 3
Subnet:
- skills-eks-a
- skills-eks-c
조건:
EC2 Node에 Public IPv4 금지
두 AZ에 분산
AWS Load Balancer Controller와 Metrics Server는 Node Group에서 실행
Flask 애플리케이션은 Node Group에서 실행 금지
Management EC2:
Name: skills-eks-admin
Subnet: skills-admin-a
Instance Type: t3.micro
Connection: SSM Session Manager
설치 도구:
AWS CLI v2
kubectl
Helm
jq
curl
다음 명령이 정상 동작해야 합니다.
aws sts get-caller-identity
aws eks update-kubeconfig --name skills-private-cluster
kubectl get nodes
kubectl get pods -A
SSH와 Public IP는 사용하지 않습니다.

제3항목: ECR과 Flask 이미지 — 10점
ECR Repository:
Repository Name: skills-flask
Tag Mutability: Immutable
Scan on Push: 활성화
Encryption: KMS
Flask 애플리케이션을 직접 작성하고 이미지로 빌드합니다.
조건:
Root 사용자 실행 금지
포트 8080
latest Tag 금지
/health는 외부 서비스와 무관하게 200
로그는 stdout/stderr 사용
Access Key를 이미지나 환경변수에 저장 금지
필수 API:
GET /health
GET /items
POST /items
GET /config
GET /health
{
 "status": "healthy"
}
POST /items
{
 "name": "keyboard",
 "price": 10000
}
응답:
{
 "id": "<생성된 ID>",
 "name": "keyboard",
 "price": 10000
}
GET /config
{
 "secret_loaded": true
}
Secret 실제 값 전체를 응답에 포함하면 안 됩니다.

제4항목: Fargate 배포 — 15점
Namespace:
skills
Fargate Profile:
Profile Name: flask-fargate-profile
Selector Namespace: skills
Kubernetes Resource:
ServiceAccount: flask-sa
Deployment: flask-app
Service: flask-service
ConfigMap: flask-config
Deployment 조건:
Replica: 2
Flask Pod는 반드시 Fargate에서 실행
Container Port: 8080
Rolling Update
Readiness Probe: /health
Liveness Probe: /health
CPU·Memory Request/Limit 설정
서로 다른 AZ에 최대한 분산
Service Type: ClusterIP
terminationGracePeriodSeconds 설정
다음 Label 또는 정보로 Fargate 실행 여부를 확인할 수 있어야 합니다.
eks.amazonaws.com/compute-type=fargate

제5항목: IAM과 IRSA — 15점
다음 Role을 분리합니다.
EKS Cluster Role
Managed Node Group Role
Fargate Pod Execution Role
Flask Application Role
Load Balancer Controller Role
Management EC2 Role
Fargate Pod Execution Role 용도:
ECR 이미지 다운로드
Fargate Pod 실행
Flask Application Role 권한:
지정 DynamoDB Table 접근
지정 Secret 조회
필요한 KMS 복호화
조건:
Flask Pod에 Access Key 저장 금지
flask-sa가 Flask Application Role 사용
Application Role과 Pod Execution Role 혼용 금지
DynamoDB Resource에 불필요한 * 금지
Secrets Manager Resource에 불필요한 * 금지
Fargate Flask 권한은 IRSA 사용
IRSA Trust Policy 제한:
Namespace: skills
ServiceAccount: flask-sa
AWS Load Balancer Controller는 별도 ServiceAccount와 별도 IAM Role을 사용합니다.

제6항목: DynamoDB·Secrets Manager — 10점
DynamoDB:
Table Name: skills-items
Partition Key: id (String)
Billing Mode: PAY_PER_REQUEST
Encryption: KMS
Point-in-Time Recovery: 활성화
Flask Application Role에는 다음 작업만 허용합니다.
dynamodb:PutItem
dynamodb:GetItem
dynamodb:Scan
Secrets Manager:
Secret Name: skills/flask/config
Encryption: KMS
예시:
{
 "environment": "competition",
 "message": "private-value"
}
Flask는 실행 시점에 Secrets Manager에서 값을 조회해야 합니다.
Kubernetes Secret 또는 이미지에 Secret 값을 복사해두는 방식은 인정하지 않습니다.

제7항목: ALB + Gateway API — 10점
AWS Load Balancer Controller를 Helm으로 설치합니다.
조건:
Controller는 system-nodegroup에서 실행
전용 ServiceAccount와 IRSA Role 사용
Flask Application Role과 분리
internet-facing ALB 생성
ALB는 Public Subnet A/C 사용
Target Type: ip
Gateway API Resource:
Gateway
HTTPRoute
HTTPRoute는 다음 경로를 Flask Service로 전달합니다.
/health
/items
/config
등록된 Target 두 개가 모두 Healthy 상태여야 합니다.
정의되지 않은 경로는 404 또는 고정 응답을 반환합니다.

제8항목: 로깅과 오토스케일링 — 5점
CloudWatch Log Group:
/skills/eks/flask
Retention: 30일
조건:
Flask stdout 로그가 CloudWatch에서 확인되어야 함
POST /items의 Method와 Path 기록
Secret 실제 값은 로그에 기록 금지
Horizontal Pod Autoscaler:
Target: flask-app
Minimum Pods: 2
Maximum Pods: 6
CPU Target: 60%
Metrics Server는 system-nodegroup에서 실행합니다.
부하 종료 후 Flask Pod가 2개로 돌아와야 합니다.

금지 사항
다음 항목 발견 시 해당 영역 감점 또는 0점 처리:
EKS Public Endpoint 활성화
Flask Pod에 Access Key 저장
Flask Pod가 Managed Node Group에서 실행
Application Role에 AdministratorAccess
모든 포트를 0.0.0.0/0에 허용
latest 이미지 사용
DynamoDB 또는 Secrets Manager Resource에 불필요한 *
Management EC2에 Public IP 또는 SSH 개방

채점표
영역
배점
VPC·Subnet·Endpoint
15
Private EKS·관리 환경
20
ECR·Flask 이미지
10
Fargate 배포
15
IAM·IRSA
15
DynamoDB·Secrets Manager
10
ALB·Gateway API
10
CloudWatch·HPA
5
합계
100

보너스 15점
WAF Rate-based Rule을 ALB에 연결: 5점
/admin/* WAF 차단: 3점
NetworkPolicy 적용: 3점
Terraform으로 VPC·EKS·Endpoint 재생성: 4점



