# GitHub Repo Transfer Automation

A Go script to **automate GitHub repository transfers** between personal accounts or organizations. Built for speed, efficiency, and convenience—especially for developers using multiple GitHub accounts (e.g., one for learning with Copilot Pro, one for actual projects).

---

## 🚀 Why This Exists

Manually transferring repositories through the GitHub UI is tedious—especially when dealing with dozens of repos. This script automates the entire process:

- **One-time setup**
- **Parallel repo transfers**
- **Retry on failure**
- **Environment-based token handling**

Built as a pure **engineering solution** that leverages **Go routines, multithreading, and API efficiency**.

---

## 🔧 Features

- **Bulk transfers**: Transfer many repositories in one go  
- **Parallelism**: Uses Go routines to run transfers concurrently  
- **HTTP optimized**: Uses [Resty](https://github.com/go-resty/resty) for fast and reliable HTTP requests  
- **Retries**: Automatic retries on failures  
- **Token management**: Reads GitHub token from `.env` file.
- **Configuration**: Uses `config.json` for specifying users and repositories.
- **Structured Logging**: Uses `logrus` for detailed and structured logging output.
- **Lightweight**: Minimal external dependencies.

---

## 📁 Setup

### 1. Clone the repo

```bash
git clone https://github.com/MishraShardendu22/Go-TransferScript.git
cd Go-TransferScript
```

### 2. Install dependencies

If you haven't already, install Go (version 1.16+ recommended). Then, fetch the dependencies:
```bash
go mod tidy
# This will download all necessary dependencies like resty, godotenv, and logrus.
# Alternatively, you can run:
# go get github.com/go-resty/resty/v2
# go get github.com/joho/godotenv
# go get github.com/sirupsen/logrus
```

### 3. Create Configuration Files

You'll need two configuration files in the root of the project:

**a) `.env` file**

This file stores your GitHub Personal Access Token.
```env
GITHUB_TOKEN_CLASSIC=your_github_token_here
```
Ensure this token has `repo` (Full control of private repositories) and `admin:repo_hook` permissions. *Note: The variable name was updated to `GITHUB_TOKEN_CLASSIC`.*

**b) `config.json` file**

This file specifies the source user, target user, and the list of repositories to transfer.
Create a `config.json` file with the following structure:

```json
{
  "originalUser": "YOUR_GITHUB_SOURCE_USERNAME",
  "newUser": "YOUR_GITHUB_TARGET_USERNAME",
  "repositories": [
    "repo-to-transfer-1",
    "another-repo",
    "my-awesome-project"
  ]
}
```
Replace the placeholder values with your actual GitHub usernames and repository names.

---

## 🚀 Run

Once the setup is complete, you can run the script:

```bash
go run main.go
```
The script will read the `.env` and `config.json` files and begin the transfer process. Progress and results will be printed to the console with structured logging.

---

## 🧠 Context

> *"I use two GitHub accounts—one for learning (Copilot Pro) and one for actual work. This script automates moving finished repos from the learning account to the project account."*

This might not be useful to everyone, but for devs managing multiple GitHub accounts or orgs, it saves **real time**. Think of it as a **batch GitHub transfer CLI**.

---

## 📣 GitHub Feature Request

I've submitted this as a feature request to GitHub.

* **[Discussion Link](https://github.com/orgs/community/discussions/163410)**
  Jump in, upvote, and add your use case if you want GitHub to support this natively.

---

## 🔮 Next Steps

If GitHub doesn’t implement this, I’ll build a minimal UI:

* OAuth + token input
* Repo fetch via API
* Bulk transfer via click

---

## 📎 Links

* 🔗 [Script Repository](https://github.com/MishraShardendu22/Go-TransferScript)
* 💬 [GitHub Discussion](https://github.com/orgs/community/discussions/163410)
