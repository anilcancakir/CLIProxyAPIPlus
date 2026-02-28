#!/usr/bin/env bash
set -euo pipefail

echo "==================================="
echo "Docker Auto-Update Cron Setup"
echo "==================================="

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)"
   exit 1
fi

# Copy the update script to /opt/
SCRIPT_SOURCE="./docker-auto-update.sh"
SCRIPT_DEST="/opt/docker-auto-update.sh"

if [[ ! -f "$SCRIPT_SOURCE" ]]; then
    echo "ERROR: $SCRIPT_SOURCE not found in current directory"
    exit 1
fi

echo "Copying docker-auto-update.sh to /opt/..."
cp "$SCRIPT_SOURCE" "$SCRIPT_DEST"
chmod +x "$SCRIPT_DEST"
echo "✓ Script installed at $SCRIPT_DEST"

# Create log file if it doesn't exist
LOG_FILE="/var/log/docker-auto-update.log"
if [[ ! -f "$LOG_FILE" ]]; then
    touch "$LOG_FILE"
    chmod 644 "$LOG_FILE"
    echo "✓ Created log file at $LOG_FILE"
fi

# Define the cron job entry
CRON_JOB="0 0,12 * * * /opt/docker-auto-update.sh >> /var/log/docker-auto-update.log 2>&1"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -Fq "/opt/docker-auto-update.sh"; then
    echo "ℹ Cron job already exists, skipping addition"
else
    echo "Adding cron job to run at 00:00 and 12:00 daily..."
    # Add the cron job (preserve existing crontab)
    (crontab -l 2>/dev/null || true; echo "$CRON_JOB") | crontab -
    echo "✓ Cron job added successfully"
fi

echo ""
echo "==================================="
echo "Setup Complete!"
echo "==================================="
echo ""
echo "The Docker auto-update script will run twice daily at:"
echo "  - 00:00 (midnight)"
echo "  - 12:00 (noon)"
echo ""
echo "Services monitored:"
echo "  - /opt/cliproxy (compose-based)"
echo "  - /opt/antigravity (compose-based)"
echo "  - /opt/kiro-gateway (git + compose)"
echo ""
echo "Logs will be written to: $LOG_FILE"
echo ""
echo "To view current cron jobs: crontab -l"
echo "To manually run the update: sudo /opt/docker-auto-update.sh"
echo "To view logs: sudo tail -f $LOG_FILE"
