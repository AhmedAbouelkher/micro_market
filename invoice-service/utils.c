#ifndef UTILS_H
#define UTILS_H

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

const char *get_env(const char *env_name, const char *default_value) {
  char *env_value = getenv(env_name);
  if (!env_value) {
    return default_value;
  }
  return env_value;
}

// Concatenate two strings
int cnstr(char *result, char const *str1, char const *str2) {
  size_t len1 = strlen(str1);
  size_t len2 = strlen(str2);
  size_t len = len1 + len2 + 1;
  int rc = snprintf(result, len, "%s%s", str1, str2);
  if (rc < 0 || (size_t)rc >= len) {
    return 1;
  }
  return 0;
}

// Get local time seconds since epoch
time_t get_local_time_seconds_since_epoch() {
  time_t now = time(NULL);
  struct tm *local_time = localtime(&now);
  return mktime(local_time);
}

int format_time(char *time_string, time_t time) {
  struct tm *local_time = localtime(&time);
  strftime(time_string, sizeof(time_string), "%Y-%m-%d %H:%M:%S", local_time);
  return 0;
}

int get_json_value(char *json_string, char *key, char *value) {
  // Build the pattern to search for: "key"
  char pattern[256];
  snprintf(pattern, sizeof(pattern), "\"%s\"", key);

  // Search for the pattern in the JSON string
  char *key_pos = strstr(json_string, pattern);
  if (!key_pos) {
    return -1;
  }

  // Move past the pattern to the character after the closing quote
  key_pos += strlen(pattern);

  // Skip whitespace
  while (*key_pos && (*key_pos == ' ' || *key_pos == '\t' || *key_pos == '\n' ||
                      *key_pos == '\r')) {
    key_pos++;
  }

  // Check for colon
  if (*key_pos != ':') {
    return -1;
  }
  key_pos++; // Move past ':'

  // Skip whitespace again
  while (*key_pos && (*key_pos == ' ' || *key_pos == '\t' || *key_pos == '\n' ||
                      *key_pos == '\r')) {
    key_pos++;
  }

  // Now key_pos should point at the start of the value
  // Consider if value is string (starts with "), number, or boolean/null
  if (*key_pos == '"') {
    // Value is a string, copy until the next unescaped quote
    key_pos++; // skip opening quote
    char *start = key_pos;
    char *out = value;
    while (*key_pos && *key_pos != '"') {
      if (*key_pos == '\\' && *(key_pos + 1)) {
        // Copy escaped character
        *out++ = *key_pos++;
      }
      *out++ = *key_pos++;
    }
    *out = '\0';
    // Optionally: Could check if *key_pos == '"' and skip
  } else {
    // Value is not a quoted string
    char *start = key_pos;
    char *out = value;
    while (*key_pos && *key_pos != ',' && *key_pos != '}' && *key_pos != '\n' &&
           *key_pos != '\r') {
      if (*key_pos == ' ')
        break; // Value finished if whitespace? (improvising for simple use)
      *out++ = *key_pos++;
    }
    *out = '\0';
    // Remove possible trailing whitespace
    int i = strlen(value) - 1;
    while (i >= 0 && (value[i] == ' ' || value[i] == '\t')) {
      value[i--] = '\0';
    }
  }

  return 0;
}

#endif