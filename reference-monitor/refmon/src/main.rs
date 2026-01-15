use chrono::Utc;
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use tonic::{transport::Server, Request, Response, Status};
use tonic_health::server::health_reporter;

// код из proto
mod refmon {
    tonic::include_proto!("refmon");
}

use refmon::reference_monitor_server::{ReferenceMonitor, ReferenceMonitorServer};
use refmon::{IssueRequest, IssueResponse, ValidateRequest, ValidateResponse};

const SECRET: &[u8] = b"super-secret-key-for-demo-2025";

#[derive(Debug, Serialize, Deserialize)]
struct Claims {
    sub: String,
    exp: usize,
}

#[derive(Default)]
pub struct RefMonImpl {}

#[tonic::async_trait]
impl ReferenceMonitor for RefMonImpl {
    async fn issue(
        &self,
        request: Request<IssueRequest>,
    ) -> Result<Response<IssueResponse>, Status> {
        let req = request.into_inner();
        let exp = (Utc::now() + chrono::Duration::seconds(req.ttl_seconds)).timestamp() as usize;

        let claims = Claims {
            sub: req.subject,
            exp,
        };

        let token = encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(SECRET),
        )
        .map_err(|e| Status::internal(e.to_string()))?;

        Ok(Response::new(IssueResponse {
            token: token.into_bytes(),
        }))
    }

    async fn validate(
        &self,
        request: Request<ValidateRequest>,
    ) -> Result<Response<ValidateResponse>, Status> {
        let token_str = String::from_utf8(request.into_inner().token)
            .map_err(|_| Status::invalid_argument("Invalid token encoding"))?;

        let validation = Validation::new(jsonwebtoken::Algorithm::HS256);
        match decode::<Claims>(&token_str, &DecodingKey::from_secret(SECRET), &validation) {
            Ok(_) => Ok(Response::new(ValidateResponse {
                valid: true,
                error: "".to_string(),
            })),
            Err(e) => Ok(Response::new(ValidateResponse {
                valid: false,
                error: e.to_string(),
            })),
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "[::]:50051".parse().unwrap();

    let refmon = RefMonImpl::default();

    // Healthcheck
    let (mut health_reporter, health_service) = health_reporter();
    health_reporter
        .set_serving::<ReferenceMonitorServer<RefMonImpl>>()
        .await;

    println!("Reference Monitor (TCB) listening on {}", addr);

    Server::builder()
        .add_service(health_service)
        .add_service(ReferenceMonitorServer::new(refmon))
        .serve(addr)
        .await?;

    Ok(())
}