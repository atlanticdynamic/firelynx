mod pdk;

use pdk::*;

#[derive(serde::Deserialize)]
struct UrlData {
    // Unused fields from go-polyscript URL structure
    // #[serde(rename = "Scheme")]
    // scheme: String,
    // #[serde(rename = "Path")]
    // path: String,
    // #[serde(rename = "Host")]
    // host: String,
    // #[serde(rename = "RawQuery")]
    // raw_query: String,
    // #[serde(rename = "Fragment")]
    // fragment: String,
}

#[derive(serde::Deserialize)]
struct RequestData {
    #[serde(rename = "Body")]
    body: String,
    
    // Unused fields from go-polyscript request structure
    // #[serde(rename = "Headers", default)]
    // headers: std::collections::HashMap<String, Vec<String>>,
    // #[serde(rename = "QueryParams", default)]
    // query_params: std::collections::HashMap<String, Vec<String>>,
    // #[serde(rename = "Method")]
    // method: String,
    // #[serde(rename = "Proto")]
    // proto: String,
    // #[serde(rename = "Host")]
    // host: String,
    // #[serde(rename = "RemoteAddr")]
    // remote_addr: String,
    // #[serde(rename = "ContentLength")]
    // content_length: i64,
    // #[serde(rename = "URL")]
    // url: UrlData,
    // #[serde(rename = "URL_Path")]
    // url_path: String,
    // #[serde(rename = "URL_Scheme")]
    // url_scheme: String,
    // #[serde(rename = "URL_Host")]
    // url_host: String,
    // #[serde(rename = "URL_String")]
    // url_string: String,
}

#[derive(serde::Deserialize)]
struct StaticData {
    search_characters: Option<String>,
    case_sensitive: Option<bool>,
    
    // Unused fields from TOML configuration
    // match_description: Option<String>,
}

#[derive(serde::Deserialize)]
struct InputData {
    request: RequestData,
    static_data: Option<StaticData>,
}

pub fn count_characters(input_json: String) -> Result<types::CharacterReport, extism_pdk::Error> {
    let input_data: InputData = serde_json::from_str(&input_json).map_err(|e| {
        extism_pdk::Error::msg(format!("Invalid JSON input: {}", e))
    })?;

    // Use static_data if available, otherwise defaults
    let matching_chars = input_data.static_data
        .as_ref()
        .and_then(|sd| sd.search_characters.as_ref())
        .map(|s| s.as_str())
        .unwrap_or("aeiouAEIOU"); // Default vowels
    
    let case_sensitive = input_data.static_data
        .as_ref()
        .and_then(|sd| sd.case_sensitive)
        .unwrap_or(false); // Default case insensitive

    // Validate character set is not empty
    if matching_chars.is_empty() {
        return Err(extism_pdk::Error::msg("Character set cannot be empty"));
    }

    // Apply case sensitivity to search text if needed
    let search_text = if case_sensitive {
        input_data.request.body.clone()
    } else {
        input_data.request.body.to_lowercase()
    };

    let target_chars = if case_sensitive {
        matching_chars.to_string()
    } else {
        matching_chars.to_lowercase()
    };

    // Count matching characters using HashSet for O(1) lookups
    let target_set: std::collections::HashSet<char> = target_chars.chars().collect();
    let count = search_text
        .chars()
        .filter(|c| target_set.contains(c))
        .count() as i32;

    Ok(types::CharacterReport {
        count,
        characters: matching_chars.to_string(),
    })
}
