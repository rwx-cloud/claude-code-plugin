#!/bin/bash
# Parse a Claude Code eval trace JSON artifact into a readable conversation summary.
# Usage: bin/parse_eval_trace.sh <artifact.json> [<artifact.json> ...]
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: $0 <artifact.json> [<artifact.json> ...]" >&2
  exit 1
fi

truncate_str() {
  local s="$1" max="${2:-200}"
  s=$(echo "$s" | tr '\n' ' ')
  local len=${#s}
  if [ "$len" -gt "$max" ]; then
    echo "${s:0:$max}... ($len chars total)"
  else
    echo "$s"
  fi
}

parse_trace() {
  local path="$1"
  echo ""
  echo "================================================================================"
  echo "Trace: $path"
  echo "================================================================================"

  # Print result summary
  jq -r '.[] | select(.type == "result") | (
    "Duration: \(.duration_ms / 1000 | tostring | split(".") | .[0] + "." + (.[1] // "0" | .[:1]))s (API: \(.duration_api_ms / 1000 | tostring | split(".") | .[0] + "." + (.[1] // "0" | .[:1]))s)",
    "Usage: \(.usage | tojson)"
  )' "$path" 2>/dev/null | while IFS= read -r line; do
    if [[ "$line" == Usage:* ]]; then
      echo "$line" | sed 's/^Usage: /Usage: /' | jq -r '"Usage:\n\(. | fromjson | to_entries[] | "  \(.key): \(.value | tojson)")"' 2>/dev/null || echo "$line"
    else
      echo "$line"
    fi
  done
  echo ""

  # Process each message
  local turn=0
  local msg_count
  msg_count=$(jq 'length' "$path")

  for ((i = 0; i < msg_count; i++)); do
    local msg_type
    msg_type=$(jq -r ".[$i].type // \"?\"" "$path")

    case "$msg_type" in
      system)
        local session_id model
        session_id=$(jq -r ".[$i].session_id // \"?\"" "$path" | cut -c1-12)
        model=$(jq -r ".[$i].model // \"?\"" "$path")
        echo "[system] session_id=$session_id model=$model"
        echo ""
        ;;

      user)
        local role body_type
        role=$(jq -r ".[$i].message.role // \"?\"" "$path")
        body_type=$(jq -r ".[$i].message.content | type" "$path")

        if [ "$body_type" = "array" ]; then
          # Tool results
          local result_count
          result_count=$(jq ".[$i].message.content | length" "$path")
          for ((j = 0; j < result_count; j++)); do
            local item_type
            item_type=$(jq -r ".[$i].message.content[$j].type // \"?\"" "$path")
            if [ "$item_type" = "tool_result" ]; then
              local result_text result_len
              result_text=$(jq -r "
                .[$i].message.content[$j].content |
                if type == \"array\" then
                  [.[] | select(type == \"object\") | .text // \"\"] | join(\" \")
                elif type == \"string\" then .
                else tostring
                end
              " "$path")
              result_len=${#result_text}
              echo "  <- tool_result ($result_len chars): $(truncate_str "$result_text")"
            fi
          done
        else
          local body body_len
          body=$(jq -r ".[$i].message.content // \"\"" "$path")
          body_len=${#body}
          echo "[$role] ($body_len chars): $(truncate_str "$body")"
        fi
        echo ""
        ;;

      assistant)
        turn=$((turn + 1))
        local content_count
        content_count=$(jq ".[$i].message.content | length" "$path")

        for ((j = 0; j < content_count; j++)); do
          local item_type
          item_type=$(jq -r ".[$i].message.content[$j].type // \"?\"" "$path")

          if [ "$item_type" = "text" ]; then
            local text
            text=$(jq -r ".[$i].message.content[$j].text // \"\"" "$path")
            if [ -n "$(echo "$text" | tr -d '[:space:]')" ]; then
              echo "[assistant turn $turn] $(truncate_str "$text" 300)"
            fi

          elif [ "$item_type" = "tool_use" ]; then
            local name
            name=$(jq -r ".[$i].message.content[$j].name // \"?\"" "$path")

            case "$name" in
              Bash)
                local cmd
                cmd=$(jq -r ".[$i].message.content[$j].input.command // \"\"" "$path")
                echo "  -> $name: $(truncate_str "$cmd" 300)"
                ;;
              Read)
                local fp
                fp=$(jq -r ".[$i].message.content[$j].input.file_path // \"\"" "$path")
                echo "  -> $name: $fp"
                ;;
              Write)
                local fp content_len
                fp=$(jq -r ".[$i].message.content[$j].input.file_path // \"\"" "$path")
                content_len=$(jq -r ".[$i].message.content[$j].input.content // \"\" | length" "$path")
                echo "  -> $name: $fp ($content_len chars)"
                ;;
              Edit)
                local fp old
                fp=$(jq -r ".[$i].message.content[$j].input.file_path // \"\"" "$path")
                old=$(jq -r ".[$i].message.content[$j].input.old_string // \"\"" "$path")
                echo "  -> $name: $fp old=$(truncate_str "$old" 80)"
                ;;
              Glob)
                local pattern gpath
                pattern=$(jq -r ".[$i].message.content[$j].input.pattern // \"?\"" "$path")
                gpath=$(jq -r ".[$i].message.content[$j].input.path // \".\"" "$path")
                echo "  -> $name: $pattern path=$gpath"
                ;;
              Grep)
                local pattern gpath
                pattern=$(jq -r ".[$i].message.content[$j].input.pattern // \"?\"" "$path")
                gpath=$(jq -r ".[$i].message.content[$j].input.path // \".\"" "$path")
                echo "  -> $name: pattern=$pattern path=$gpath"
                ;;
              *)
                local inp
                inp=$(jq -c ".[$i].message.content[$j].input // {}" "$path")
                echo "  -> $name: $(truncate_str "$inp" 200)"
                ;;
            esac
          fi
        done
        echo ""
        ;;
    esac
  done

  echo "Total messages: $msg_count, Assistant turns: $turn"
}

for path in "$@"; do
  parse_trace "$path"
done
