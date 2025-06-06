# ---------- 基於官方 Gotenberg v8 映像 ----------
FROM gotenberg/gotenberg:8

# ---------- 維護者資訊（選填，可自行更改為你名稱/Email） ----------
LABEL maintainer="Your Name <youremail@example.com>"

# ---------- 設定 Gotenberg 關鍵環境變數 ----------
ENV API_BODY_LIMIT="100MB" \
    DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE="104857600" \
    DEFAULT_WAIT_TIMEOUT="15" \
    MAXIMUM_WAIT_TIMEOUT="60"

# ---------- 覆蓋預設啟動命令（若要使用預設可刪除這段） ----------
CMD ["gotenberg", "--api-body-limit=100MB", "--default-wait-timeout=15s", "--max-wait-timeout=60s"]
