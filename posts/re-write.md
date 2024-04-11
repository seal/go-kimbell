# Intro
When I initially wrote the code for this site, it was to create a better solution than something like [Hugo](https://gohugo.io/) but after a day of bashing up my first http-server in Rust, it was botched, and not my best work, but it *worked*

Using axum, Postgres & static HTML it lacked many things, but longevity was my main concern.

For example, to create a new post I had to open up post-man, due to my lack of knowledge of tools such as clap, and had no containerization.

It wasn't my best work.

## Design 

![https://img.kimbell.uk/flameshot_image_t3ho4q.png](https://img.kimbell.uk/flameshot_image_t3ho4q.png)

Simple yet effective, Docker & Nginx behind Tide.
![https://img.kimbell.uk/flameshot_image_y5zfxa.png](https://img.kimbell.uk/flameshot_image_y5zfxa.png)
The CLI-tool, the same again, simply a new post option, or delete

## Tech Stack
After being inspired by a DreamsOfCode [video](https://www.youtube.com/watch?v=ZbhzLP3vnkg),I decided on [Tide](https://github.com/http-rs/tide) for my http-server for it's seemingly go-like syntax and inner-workings.
Leaving me with:
- [Tide](https://github.com/http-rs/tide) 
- [Vue](https://vuejs.org/)
- Sqlite3
- [Sqlx](https://github.com/launchbadge/sqlx)
- Docker

## Issues
There was no lack of issues I encountered, varying with frustration as my project went along...
#### UUID's 
With my sqlx migration being this file:
```sql
-- migrations/1_initial_setup.sql

CREATE TABLE posts (
    id UUID NOT NULL PRIMARY KEY,
    title TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
```
And my model being:
```rust
use sqlx::types::Uuid;
#[derive(sqlx::FromRow, Serialize)]
pub struct Post {
    pub id: Uuid,
    pub title: String,
    pub created_at: i64,
}
```
When creating a post via:
```rust
        let inserted_post = sqlx::query_as!(
    Post,
    r#"INSERT INTO posts (id, title, created_at) VALUES (?, ?, ?) RETURNING id, title, created_at"#,
    id,
    new_post.title,
    new_post.created_at
)
```
I encountered a sqlx error:
```
# `unsupported type NULL of column #1`
```
After some digging I eventually sorted this issue via [this GitHub issue](https://github.com/launchbadge/sqlx/issues/1350)

- This would later bite me in the ass, as UUID's were being store inside the database as "OK\[I3%", so I would revert this.
#### Docker...
Containerizing my code was not a huge issue, but *optimizing it* proved to be a much harder challenge, with 5 minute builds locally, and missing dependencies left right and center, I had to battle with docker for many hours, and eventually settled on this:
```Dockerfile
# Use a smaller base image for the builder stage
FROM rust:slim as builder

WORKDIR /app

COPY . .
COPY .env .

# Install build dependencies
RUN apt-get update && \
    apt-get install -y pkg-config libssl-dev perl make musl-tools 

# Add only the necessary target for the build
RUN rustup target add x86_64-unknown-linux-musl

# Build the application
RUN cargo build --release --target=x86_64-unknown-linux-musl && \
    strip target/x86_64-unknown-linux-musl/release/kimbell


# Use a smaller base image for the final stage
FROM alpine:3.14

WORKDIR /app

# Copy only the necessary files from the builder stage
COPY --from=builder /app/target/x86_64-unknown-linux-musl/release/kimbell /app/
COPY database.sqlite /app/
COPY dist /app/dist
COPY md_files /app/md_files

EXPOSE 3000

CMD ["./kimbell"]

```
Whilst more optimizations could take place, my sanity might be lost in the meantime.

Coming from Go, and it being my primary, and only language I would describe myself as advanced in from 2018-2020, combining rust's slow builds + a docker build step is excruciating to say the least..


#### Nginx's headers
I've setup my fair share of static JS sites using Nginx, or using the reverse-proxy feature to send requests to an Express JS server. One thing I had never used, due to the simplicity of the site's I have made, was headers.

After creating my CLI-tool to create and delete posts, the API-hey in the header "api-key" was not working....
After many hours of confusion, googling and frustration, I realized NGINX removes headers with \_'s in by default... 

I promptly changed all occurrences of said underscore, then promptly forgot *one* of them and continued to debug for hours, exciting...


## Code

Main.rs:
```rust
use crate::handlers::{create_post, delete_post, get_post, get_posts};
use dotenv::dotenv;
use std::io::ErrorKind;
use tide::log::error;
use tide::log::info;
use tide::log::LevelFilter;
use tide::utils::After;
mod handlers;
mod models;
use env_logger;
use sqlx::sqlite::SqlitePoolOptions;
use std::env;
use tide::Response;
use tide::StatusCode;

#[derive(Clone)]
pub struct State {
    db_pool: sqlx::SqlitePool,
}

#[async_std::main]
async fn main() -> tide::Result<()> {
    env_logger::builder()
        .filter_level(LevelFilter::Debug)
        .init();

    info!("Initialising...");
    if let Err(e) = async_std::fs::create_dir_all("./md_files").await {
        eprintln!("Error creating 'md_files' directory: {}", e);
        return Err(tide::Error::from_str(
            StatusCode::InternalServerError,
            "Internal Server Error",
        ));
    }
    dotenv().ok();
    info!("Env vars OK");
    let database_url = env::var("DATABASE_URL").expect("DB URL NOT SET");
    let db_pool = SqlitePoolOptions::new()
        .max_connections(5)
        .connect(&database_url)
        .await?;
    let state = State { db_pool };
    let mut app = tide::with_state(state);
    app.with(tide::log::LogMiddleware::new());
    info!("DB Connection OK");
    app.with(After(|mut res: Response| async {
        if let Some(err) = res.downcast_error::<async_std::io::Error>() {
            match err.kind() {
                ErrorKind::NotFound => {
                    error!("{:?}", err);
                    let msg = format!("Error: {:?}", err);
                    res.set_status(StatusCode::NotFound);
                    res.set_body(msg);
                }
                _ => {
                    error!("{:?}", err);
                    let msg = format!("Internal Server Error: {:?}", err);
                    res.set_status(StatusCode::InternalServerError);
                    res.set_body(msg);
                }
            }
        }
        Ok(res)
    }));

    app.at("/api/posts").get(get_posts);
    app.at("/api/posts").post(create_post);
    app.at("/api/posts/:uuid").delete(delete_post);
    app.at("/api/post/:id").get(get_post);
    app.at("/posts").serve_file("dist/index.html")?;
    app.at("/post/:id").serve_file("dist/index.html")?;
    app.at("/").serve_file("dist/index.html")?;
    app.at("/index.html").serve_file("dist/index.html")?;
    app.at("/assets/").serve_dir("dist/assets/")?;
    info!("Created Routes");
    app.listen("0.0.0.0:3000").await?;
    Ok(())
}
```

handlers.rs
```rust
pub async fn get_posts(req: Request<State>) -> tide::Result {
    let db_pool = &req.state().db_pool;

    let posts = query_as!(Post, r#"SELECT id, title, created_at FROM posts"#)
        .fetch_all(db_pool)
        .await?;

    let response_json = json!({
        "info": { "count": posts.len() },
        "posts": posts,
    });

    let mut response = Response::new(StatusCode::Ok);
    response.insert_header("Content-Type", "application/json");
    response.insert_header("Access-Control-Allow-Origin", "*");
    response.set_body(response_json);

    Ok(response)
}
pub async fn get_post(req: Request<State>) -> tide::Result {
    let md_file_name: String = req.param("id")?.to_string();
    let db_pool = &req.state().db_pool;
    let id = md_file_name.clone();
    let post = query_as!(
        PostNoID,
        r#"SELECT title, created_at FROM posts WHERE id=?"#,
        id
    )
    .fetch_one(db_pool)
    .await?;

    let file_content = read_md_file(&md_file_name).await?;

    let post_content = parse_md_content(&file_content);

    let response_json = json!({
        "post": NewPost{
            created_at: post.created_at,
            title: post.title,
            content: post_content.content
        },
    });

    let mut response = Response::new(StatusCode::Ok);
    response.insert_header("Content-Type", "application/json");
    response.insert_header("Access-Control-Allow-Origin", "*");
    response.set_body(response_json);

    Ok(response)
}

async fn read_md_file(file_name: &str) -> tide::Result<String> {
    let file_path = format!("./md_files/{}.md", file_name);
    match async_std::fs::read_to_string(file_path).await {
        Ok(content) => Ok(content),
        Err(e) => {
            let error_msg = format!("Error reading MD file: {} : {}", file_name, e);
            Err(tide::Error::new(
                StatusCode::NotFound,
                anyhow::Error::msg(error_msg),
            ))
        }
    }
}

fn parse_md_content(content: &str) -> PostContent {
    PostContent {
        content: content.to_string(),
    }
}
async fn store_md_file(id: &String, content: &str) -> tide::Result<()> {
    let file_path = format!("./md_files/{}.md", id);
    fs::write(&file_path, content)?;

    Ok(())
}

pub async fn delete_post(req: Request<State>) -> tide::Result {
    let db_pool = &req.state().db_pool;
    let api_key = env::var("API_KEY").expect("API_KEY not found in .env");
    let provided_key: String = req
        .header("api-key")
        .map(|header_values| header_values.as_str().to_string())
        .unwrap_or_else(|| "na".to_string());
    if provided_key == api_key {
        let uuid_param: String = req.param("uuid").unwrap_or("Error").to_string();
        if sqlx::query!(r#"DELETE FROM posts WHERE id = ?"#, uuid_param)
            .execute(db_pool)
            .await?
            .rows_affected()
            > 0
        {
            if let Err(file_error) = fs::remove_file(format!("./md_files/{}.md", uuid_param)) {
                info!(
                    "Error deleting file: ./md_files/{}.md Error : {}",
                    uuid_param, file_error
                );
                let response_json = json!({
                    "message": "Error deleting file",
                    "error": file_error.to_string(),
                });
                let mut response = Response::new(StatusCode::InternalServerError);
                response.insert_header("Content-Type", "application/json");
                response.insert_header("Access-Control-Allow-Origin", "*");
                response.set_body(response_json);

                return Ok(response);
            }

            info!("Successfully deleted post with ID {uuid_param}");
            let response_json = json!({
                "message": "Post deleted successfully",
            });

            let mut response = Response::new(StatusCode::Ok);
            response.insert_header("Content-Type", "application/json");
            response.insert_header("Access-Control-Allow-Origin", "*");
            response.set_body(response_json);

            return Ok(response);
        } else {
            info!("Unsuccessfully deleted post with ID {uuid_param}");
            // Handle the case where the post was not found
            let response_json = json!({
                "message": "Post not found",
            });

            let mut response = Response::new(StatusCode::NotFound);
            response.insert_header("Content-Type", "application/json");
            response.insert_header("Access-Control-Allow-Origin", "*");
            response.set_body(response_json);

            return Ok(response);
        }
    }

    let response_json = json!({
        "message": "Unauthorized. Invalid API key",
    });

    let mut response = Response::new(StatusCode::Unauthorized);
    response.insert_header("Content-Type", "application/json");
    response.insert_header("Access-Control-Allow-Origin", "*");
    response.set_body(response_json);

    Ok(response)
}

pub async fn create_post(mut req: Request<State>) -> tide::Result {
    let api_key = env::var("API_KEY").expect("API_KEY not found in .env");

    let provided_key: String = req
        .header("api-key")
        .map(|header_values| header_values.as_str().to_string())
        .unwrap_or_else(|| "na".to_string());
    if provided_key == api_key {
        let new_post: NewPost = req.body_json().await?;
        info!("Received new post - Name: {}", new_post.title);
        let db_pool = &req.state().db_pool;
        let id = Uuid::new_v4();
        let id_string = id.to_string();
        info!("UUID Created {}", id_string);
        store_md_file(&id_string, &new_post.content).await?;
        let inserted_post = sqlx::query_as!(
    Post,
    r#"INSERT INTO posts (id, title, created_at) VALUES (?, ?, ?) RETURNING id, title, created_at"#,
    id_string,
    new_post.title,
    new_post.created_at
)
    .fetch_one(db_pool)
    .await?;
        let response_json = json!({
            "message": "Post created successfully",
            "post": inserted_post,
        });

        let mut response = Response::new(StatusCode::Created);
        response.insert_header("Content-Type", "application/json");
        response.insert_header("Access-Control-Allow-Origin", "*");
        response.set_body(response_json);

        return Ok(response);
    }

    let response_json = json!({
        "message": "Unauthorized. Invalid API key",
    });

    let mut response = Response::new(StatusCode::Unauthorized);
    response.insert_header("Content-Type", "application/json");
    response.insert_header("Access-Control-Allow-Origin", "*");
    response.set_body(response_json);

    Ok(response)
}
```



new_post.rs ( CLI tool):
```rust
use anyhow::{anyhow, Ok};
use reqwest::Client;
use reqwest::{self};
use serde::{Deserialize, Serialize};
use std::fs::File;
use std::io::Read;
use std::path::PathBuf;
use std::{env, io};
mod models;
use crate::models::ApiResponse;
use clap::{arg, command, value_parser, Command};

#[tokio::main]
async fn main() -> Result<(), anyhow::Error> {
    dotenv::dotenv().ok();

    let matches = command!()
        .about("Kimbell Posts - CLI to edit posts on my site")
        .version("0.0.1")
        .subcommand(
            Command::new("create")
                .about("Creates a post")
                .arg(
                    arg!(
                        -n --name <String> "Sets the name"
                    )
                    .required(true)
                    .value_parser(clap::value_parser!(String)),
                )
                .arg(
                    arg!(
                        -f --file <FILE> "Sets the post file"
                    )
                    .required(true)
                    .value_parser(value_parser!(std::path::PathBuf)),
                ),
        )
        .subcommand(
            Command::new("delete").about("Deletes a post").arg(
                arg!(
                    -u --uuid <String> "Sets the UUID"
                )
                .required(false),
            ),
        )
        .get_matches();

    match matches.subcommand_name() {
        Some("create") => {
            let create_matches = matches.subcommand_matches("create").unwrap();
            create_post(
                create_matches
                    .get_one::<String>("name")
                    .unwrap()
                    .to_string(),
                create_matches
                    .get_one::<PathBuf>("file")
                    .unwrap()
                    .to_path_buf(),
            )
            .await?;
        }
        Some("delete") => {
            let delete_matches = matches.subcommand_matches("delete").unwrap();
            delete_post(delete_matches.get_one::<String>("uuid").cloned()).await?;
        }
        _ => println!("No subcommand provided"),
    }
    Ok(())
}
#[derive(Serialize, Deserialize)]
struct NewPost {
    content: String,
    created_at: i64,
    title: String,
}
async fn create_post(name: String, path: PathBuf) -> Result<(), anyhow::Error> {
    println!("Creating post with name {} and path {:?}", name, path);
    let api_key = env::var("API_KEY").expect("API_KEY not found in .env");
    let local = env::var("LOCAL").expect("LOCAL not found in .env");
    let mut file = File::open(&path)?;
    let mut content = String::new();
    file.read_to_string(&mut content)?;

    let date = chrono::Utc::now().timestamp();

    let my_json_struct = NewPost {
        content,
        created_at: date,
        title: name,
    };

    let json_body = serde_json::to_string(&my_json_struct)?;

    let client = Client::new();
    let uri = match local.as_str() {
        "true" => "http://127.0.0.1:3000".to_string(),
        "false" => "https://kimbell.uk".to_string(),
        _ => "error".to_string(),
    };
    let response = client
        .post(format!("{uri}/api/posts"))
        .header("api-key", api_key)
        .body(json_body)
        .send()
        .await?;

    if response.status().is_success() {
        println!("File successfully uploaded!");
        Ok(())
    } else {
        Err(anyhow!(
            "Failed to upload file. Status code: {}, body {}",
            response.status(),
            response.text().await?
        )
        .into())
    }
}

async fn delete_post(uuid: Option<String>) -> Result<(), anyhow::Error> {
    match uuid {
        Some(uuid) => {
            println!("Deleting post {}", &uuid);
            delete_post_by_uuid(&uuid).await
        }
        None => {
            let api_key = env::var("API_KEY").expect("API_KEY not found in .env");
            let client = Client::new();
            let local = env::var("LOCAL").expect("LOCAL not found in .env");
            let uri = match local.as_str() {
                "true" => "http://127.0.0.1:3000".to_string(),
                "false" => "https://kimbell.uk".to_string(),
                _ => return Err(anyhow!("Invalid value for 'LOCAL' in .env")),
            };

            let api_response = client
                .get(format!("{}/api/posts", uri))
                .header("api-key", &api_key)
                .send()
                .await?
                .json::<ApiResponse>()
                .await?;

            let uuids: Vec<(String, String)> = api_response
                .posts
                .iter()
                .map(|post| (post.id.clone(), post.title.clone()))
                .collect();

            if let Some((selected_uuid, selected_title)) = choose_uuid(&uuids) {
                println!("Deleting post {}: {}", &selected_uuid, &selected_title);
                delete_post_by_uuid(&selected_uuid).await
            } else {
                println!("No UUID selected. Exiting.");
                Ok(())
            }
        }
    }
}

async fn delete_post_by_uuid(uuid: &str) -> Result<(), anyhow::Error> {
    let api_key = env::var("API_KEY").expect("API_KEY not found in .env");
    let client = Client::new();
    let local = env::var("LOCAL").expect("LOCAL not found in .env");
    let uri = match local.as_str() {
        "true" => "http://127.0.0.1:3000".to_string(),
        "false" => "https://kimbell.uk".to_string(),
        _ => return Err(anyhow!("Invalid value for 'LOCAL' in .env")),
    };

    let response = client
        .delete(format!("{}/api/posts/{}", uri, uuid))
        .header("api-key", &api_key)
        .send()
        .await?;

    if response.status().is_success() {
        println!("File with UUID {} successfully deleted!", uuid);
        Ok(())
    } else {
        Err(anyhow!("Failed to delete file. Status code: {}", response.status()).into())
    }
}

fn choose_uuid(uuids: &[(String, String)]) -> Option<(String, String)> {
    println!("Available UUIDs:");
    for (index, (uuid, title)) in uuids.iter().enumerate() {
        println!("{}. {}: {}", index + 1, uuid, title);
    }

    println!("Enter the number corresponding to the UUID you want to delete:");
    let mut input = String::new();
    io::stdin().read_line(&mut input).ok()?;

    if let anyhow::Result::Ok(index) = input.trim().parse::<usize>() {
        if index > 0 && index <= uuids.len() {
            return Some(uuids[index - 1].clone());
        }
    }

    println!("Invalid selection. Please enter a valid number.");
    None
}
```

There are more files on the [Repository Link](https://github.com/seal/kimbell.uk) but for length's sake, I'll keep it to these two files.



## Conclusion 
Moving from go->rust has been, interesting...
With incredible compiler errors giving me suggestions that tell me *exactly* what I need to do, to compiler errors with little to no documentation on the internet, I find myself in a love-hate relationship with Rust.

It's been a great experience to code in this language, and I will choose it for many future projects, except more-so on the large-scale ones, as where I am currently at, my speed at development is far higher in Go than Rust.

Tide's simple nature is amazing, although I believe a more consistent method for database's needs to be adopted across http-server frameworks.

