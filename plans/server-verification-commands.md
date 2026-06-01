# Juvia Panel - Server Verification Commands

## SSH into your server
```bash
ssh root@103.143.0.169
```

---

## 1. Check Installed Caddyfile
```bash
cat /etc/panel/caddy/Caddyfile
```
Expected content should match the source at `backend/config/Caddyfile`:
- Port 2053 binding
- Proxy `/api/*` to `localhost:9090`
- `try_files {path} /index.html` for SPA fallback

---

## 2. Check Caddy Service Status
```bash
systemctl status juvia-caddy
```
- Should show "active (running)"
- Check for any error messages

---

## 3. Restart Caddy to Load New Config
```bash
systemctl restart juvia-caddy
systemctl status juvia-caddy
```

---

## 4. Test API Direct Access (Port 9090)
```bash
curl http://localhost:9090/api/v1/health
```
Expected: `{"status":"ok"}` or similar JSON response

---

## 5. Test API Through Caddy (Port 2053)
```bash
curl http://localhost:2053/api/v1/health
```
Expected: Same JSON response from API (proxied through Caddy)

---

## 6. Test UI Root
```bash
curl -s http://localhost:2053/ | head -20
```
Expected: HTML content of the index page

---

## 7. Check Caddy Error Logs
```bash
journalctl -u juvia-caddy -n 50 --no-pager
```

---

## 8. Validate Caddy Config Syntax
```bash
caddy validate --config /etc/panel/caddy/Caddyfile --adapter caddyfile
```

---

## 9. Check Caddy Process
```bash
ps aux | grep caddy
```

---

## 10. Test Static File Access
```bash
curl -I http://localhost:2053/_next/static/chunks/main.js 2>&1 | head -5
```

---

## If Issues Found

### Caddy not starting?
```bash
caddy run --config /etc/panel/caddy/Caddyfile --adapter caddyfile
```
Watch for error messages.

### API not responding?
```bash
systemctl status juvia-api
journalctl -u juvia-api -n 20 --no-pager
```

### Files not found (404)?
```bash
ls -la /opt/panel/ui/.next/server/app/
```
Ensure the Next.js build exists at this path.