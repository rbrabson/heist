# heist

A work-in-progress to learn more about Discord bots and MongoDB, using the Go programming language.

## Run Docker Image

### Build Container

``` bash
docker-buildx build -t heist:1.0.0 .
docker push heist:1.0.0
```
### Start Container

```bash
docker run --env BOT_TOKEN="<bot-token>" --env APP_ID="<app-id>" --env HEIST_DEFAULT_THEME="<theme-name>" --env HEIST_THEME_DIR="<theme-dir>" --name <container-name> heist:1.0.0
```