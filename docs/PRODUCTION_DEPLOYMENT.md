# Production Deployment Guide

## 1. Server Prerequisites
- A Linux VPS (Ubuntu 22.04 recommended).
- Git, Go (1.20+), Node.js (18+), and Nginx installed.
- MongoDB Atlas (Cloud) connection string.
- A domain name (e.g., `YOUR_DOMAIN`) pointing to `YOUR_SERVER_IP`.

## 2. Go Installation & Build
```bash
git clone https://github.com/your-repo/botupdate_go.git /opt/botupdate
cd /opt/botupdate
go build -o botupdate ./cmd/bot
```

## 3. Node Build
```bash
cd web
npm install
npm run build
```
*(The Go backend serves the `web/dist` folder automatically on `/`)*

## 4. MongoDB Configuration
Whitelist your VPS IP `YOUR_SERVER_IP` in MongoDB Atlas Network Access.

## 5. Environment Variables
Create `/opt/botupdate/.env`:
```env
MONGO_URL=YOUR_MONGO_URI
TELEGRAM_BOT_TOKEN=YOUR_BOT_TOKEN
API_ID=YOUR_API_ID
API_HASH=YOUR_API_HASH
PORT=3000
```
*(Only supply WEB_ADMIN_USERNAME/PASSWORD on the very first boot to bootstrap the owner account)*

## 6. Systemd Setup
Copy the service file:
```bash
sudo cp docs/systemd/botupdate.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable botupdate
sudo systemctl start botupdate
```

## 7. Nginx Configuration
```bash
sudo cp docs/nginx-production.conf /etc/nginx/sites-available/botupdate
sudo ln -s /etc/nginx/sites-available/botupdate /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

## 8. SSL Setup (Certbot)
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d YOUR_DOMAIN
```

## 9. Firewall Ports
Ensure ports 80, 443, and 22 are open:
```bash
sudo ufw allow 'Nginx Full'
sudo ufw allow OpenSSH
sudo ufw enable
```

## 10. Verification
- Navigate to `https://YOUR_DOMAIN/health` -> `{"status":"ok"}`
- Navigate to `https://YOUR_DOMAIN/ready` -> `{"status":"ready"}`
- Login to the web dashboard to verify WebSocket connects without mixed-content errors.

## 11. Logs & Restart
- Logs: `sudo journalctl -u botupdate -f`
- Restart: `sudo systemctl restart botupdate`

## 12. Backup & Rollback
See `PRODUCTION_BACKUP.md` for MongoDB backups. For application rollback, re-compile the previous git tag and restart the systemd service.
