Having just re-written this site, I needed a way to host images, which I quickly realized after uploading my first post, only to be greeted with \!\[\[Pasted Image XXXXX\]\] from my use of obsidian.

Repo link:
[Link](https://github.com/seal/img.kimbell.uk)

First off we start with a basic rust project, using 
```
cargo new image-tool
```

Then running 
```
cargo add dotenv env_logger tide 
cargo add async-std --features="attributes"
cargo add tokio --features="full"
```
for our dependencies .

Our main.rs will look like this:
```rust
use crate::handlers::upload;
use dotenv::dotenv;
use std::io::ErrorKind;
use tide::log::error;
use tide::log::info;
use tide::log::LevelFilter;
use tide::utils::After;
mod handlers;
use env_logger;
use tide::Response;
use tide::StatusCode;

#[async_std::main]
async fn main() -> tide::Result<()> {
    env_logger::builder()
        .filter_level(LevelFilter::Debug)
        .init();

    info!("Initialising...");
    if let Err(e) = async_std::fs::create_dir_all("./images").await {
        eprintln!("Error creating 'images' directory: {}", e);
        return Err(tide::Error::from_str(
            StatusCode::InternalServerError,
            "Internal Server Error",
        ));
    }
    dotenv().ok();
    std::env::var("DOMAIN").expect("No domain env variable set");
    std::env::var("API_KEY").expect("No api-key env variable set");
    info!("Env vars OK");
    let mut app = tide::new();
    app.with(tide::log::LogMiddleware::new());
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

    app.at("/new/:file").put(upload);
    app.at("/").serve_dir("images/")?;
    info!("Created Routes");
    app.listen("0.0.0.0:3001").await?;
    Ok(())
}

```

A route for file upload, and an endpoint to serve images.

Our handlers.rs will look like this:
```rust
use async_std::fs::OpenOptions;
use async_std::io;
use tide::prelude::json;
use tide::{log::info, Request};
use tide::{Response, StatusCode};

/// Handles image upload.
pub async fn upload(req: Request<()>) -> tide::Result {
    // Retrieve API key from environment variables
    let api_key = std::env::var("API_KEY").expect("API_KEY not found in .env");

    // Retrieve the provided API key from the request headers
    let provided_key: String = req
        .header("API-KEY")
        .map(|header_values| header_values.as_str().to_string())
        .unwrap_or_else(|| "na".to_string());

    // Check if the provided API key is valid
    if provided_key == api_key {
        // Extract file path from the request parameters
        let path = req.param("file")?.to_string().clone();

        // Build the file system path
        let fs_path = format!("./images/{}", path);

        // Open the file for writing
        let file = OpenOptions::new()
            .create(true)
            .write(true)
            .open(&fs_path)
            .await?;

        // Copy the request body (file content) to the opened file
        let bytes_written = io::copy(req, file).await?;

        // Log information about the uploaded file
        info!("file written", {
            bytes: bytes_written,
            path: fs_path,
        });

        // Retrieve domain from environment variables
        let domain = std::env::var("DOMAIN").expect("No domain env variable set");

        // Create a JSON response with the generated image URL
        let mut response = Response::new(StatusCode::InternalServerError);
        response.insert_header("Content-Type", "application/json");
        response.insert_header("Access-Control-Allow-Origin", "*");
        response.set_body(json!({
            "file": format!("{}/{}", domain, path),
        }));

        return Ok(response);
    }

    // Handle unauthorized access with a JSON response
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

We will be using an api-key from a .env file to avoid too many complications.

### Bash
Now that our code is written, calling said image-api is slightly more complicated.

Currently I use flameshot GUI to take images and then I copy to clipboard, I wanted to retain this functionality so I wrote an upload script to do this for me.


Steps:
- Open flameshot GUI 
- We take our screenshot
- Copy to clipboard
- Create a temporary file 
- Curl "put" our file to our domain
- Copy the json response to our clipboard using jq to get the "file" attribute

Here's the finished product:
```bash
#!/bin/bash

if ! command -v xclip &> /dev/null; then
    notify-send "Error: xclip is not installed. Please install it first."
    exit 1
fi

flameshot_cmd="flameshot gui"
$flameshot_cmd

temp_file=$(mktemp /tmp/XXXXXX.png)
xclip -selection clipboard -t image/png -o > "$temp_file"

if [ ! -s "$temp_file" ]; then
    notify-send "Error: Temporary file does not contain an image. Exiting."
    rm "$temp_file"  
    exit 1
fi

upload_url="PLACEHOLDER_DOMAIN"
api_key="PLACEHOLDER_API_KEY"

response=$(curl -s -H "API-KEY: $api_key" -T "$temp_file" "$upload_url/new/")
image_url=$(echo "$response" | jq -r '.file')

echo -n "$image_url" | xclip -selection clipboard

notify-send "Screenshot Uploaded" "Server Response copied to clipboard."

rm "$temp_file"
```

Now we need to create our .env file with two variables:
```
DOMAIN=https://DOMAIN.COM
API_KEY=API-KEY-HERE
```

Then for other's ease of use, we create a setup script so they can easily modify the upload script.

```bash
#!/bin/bash

# Read the domain from .env file
if [ -f .env ]; then
    source .env
else
    echo "Error: .env file not found."
    exit 1
fi

# Set the domain & api-key in upload.sh
sed -i "s|PLACEHOLDER_DOMAIN|$DOMAIN|" ./upload.sh
sed -i "s|PLACEHOLDER_API_KEY|$API_KEY|" ./upload.sh

echo "Domain & api-key set in upload.sh: $DOMAIN , $API_KEY"
```

Now we run:
```
chmod +x upload.sh && chmod +x setup.sh
```

After uploading our code to our server, in my case https://img.kimbell.uk , we can run the setup script and start editing our i3 config.

### Adding to i3 

```
cp ./upload.sh ~/.config/i3/
```

Added to ~/.config/i3/config:
```
bindsym $mod+a exec --no-startup-id ~/.config/i3/upload.sh
```

Then with a re-loading of our i3 config, when I take a screenshot and copy to clipboard, our image will be uploaded and our URL copied to clipboard.

### Nginx file-size
Nginx has a default limit of size, that you need to alter to allow for large-photo's to be sent.

Inside:
```
/etc/nginx/nginx.conf
```
Add this inside the http block:
```
client_max_body_size 100M;
```

Now this is done, we can upload large-images like this :) 

![https://img.kimbell.uk/flameshot_image_8Q5MBX.png](https://img.kimbell.uk/flameshot_image_8Q5MBX.png)


All in all this was a fairly quick program to write, this is my second application using [Tide](https://github.com/http-rs/tide) and it will not be my last.
