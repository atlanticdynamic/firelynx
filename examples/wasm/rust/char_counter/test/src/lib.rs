use extism_pdk::*;
use xtp_test;
use serde_json::json;

#[derive(serde::Serialize, serde::Deserialize)]
pub struct CharacterReport {
    count: i32,
    characters: String,
}

// Helper function to create realistic test input matching go-polyscript format
fn create_test_input(body: &str) -> String {
    create_test_input_with_config(body, None, None)
}

// Helper function to create test input with static_data configuration
fn create_test_input_with_config(body: &str, search_chars: Option<&str>, case_sensitive: Option<bool>) -> String {
    let mut input = json!({
        "request": {
            "Body": body,
            "Headers": {
                "Content-Type": ["application/json"],
                "User-Agent": ["xtp-test/1.0"]
            },
            "QueryParams": {},
            "Method": "POST",
            "Proto": "HTTP/1.1",
            "Host": "localhost:8080",
            "RemoteAddr": "[::1]:12345",
            "ContentLength": body.len(),
            "URL": {
                "Scheme": "http",
                "Path": "/api/demo",
                "Host": "localhost:8080",
                "RawQuery": "",
                "Fragment": ""
            },
            "URL_Path": "/api/demo",
            "URL_Scheme": "http",
            "URL_Host": "localhost:8080",
            "URL_String": "/api/demo"
        }
    });

    // Add static_data if configuration is provided
    if search_chars.is_some() || case_sensitive.is_some() {
        let mut static_data = json!({});
        if let Some(chars) = search_chars {
            static_data["search_characters"] = json!(chars);
        }
        if let Some(case_sens) = case_sensitive {
            static_data["case_sensitive"] = json!(case_sens);
        }
        input["static_data"] = static_data;
    }

    input.to_string()
}

#[plugin_fn]
pub fn test() -> FnResult<()> {
    // Test with mock input if provided by test harness
    if let Some(mock_data) = xtp_test::mock_input::<String>() {
        let Json(result): Json<CharacterReport> = xtp_test::call("CountCharacters", mock_data)?;
        xtp_test::assert_ne!("mock input produces result", &result.characters, "");
    }

    // Basic functionality tests
    let input = create_test_input("Hello World");
    let Json(result): Json<CharacterReport> = xtp_test::call("CountCharacters", &input)?;
    xtp_test::assert_eq!("Hello World has 3 vowels", result.count, 3);
    xtp_test::assert_eq!("Uses default vowel set", &result.characters, "aeiouAEIOU");

    // Edge case: empty input
    let empty_input = create_test_input("");
    let Json(empty_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &empty_input)?;
    xtp_test::assert_eq!("Empty string has 0 vowels", empty_result.count, 0);

    // Edge case: no vowels
    let no_vowels_input = create_test_input("xyz");
    let Json(no_vowels_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &no_vowels_input)?;
    xtp_test::assert_eq!("xyz has no vowels", no_vowels_result.count, 0);

    // All vowels test
    let all_vowels_input = create_test_input("aeiou");
    let Json(all_vowels_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &all_vowels_input)?;
    xtp_test::assert_eq!("aeiou has 5 vowels", all_vowels_result.count, 5);

    // Case sensitivity test (default is case-insensitive)
    let mixed_case_input = create_test_input("HELLO world");
    let Json(mixed_case_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &mixed_case_input)?;
    xtp_test::assert_eq!("HELLO world has 3 vowels (case insensitive)", mixed_case_result.count, 3);

    // Performance test - measure execution time
    let large_input = create_test_input(&"aeiou".repeat(1000));
    let time_ns = xtp_test::time_ns("CountCharacters", &large_input)?;
    xtp_test::assert_lt!("large input processes quickly", time_ns, 1e8 as u64); // < 100ms

    // Test complex JSON content
    xtp_test::group("JSON content tests", || {
        let json_string_input = create_test_input("\"Hello World\"");
        let Json(json_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &json_string_input)?;
        xtp_test::assert_eq!("JSON string has 3 vowels", json_result.count, 3);

        let complex_json = r#"{"message":"Hello World","active":true}"#;
        let complex_input = create_test_input(complex_json);
        let Json(complex_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &complex_input)?;
        
        // Count expected vowels in the JSON content
        let expected = complex_json.chars().filter(|c| "aeiouAEIOU".contains(*c)).count() as i32;
        xtp_test::assert_eq!("complex JSON vowel count", complex_result.count, expected);
        
        Ok(())
    })?;

    // Test various input types
    xtp_test::group("input variety tests", || {
        // Numbers and special characters
        let mixed_input = create_test_input("123!@#aeiou$%^");
        let Json(mixed_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &mixed_input)?;
        xtp_test::assert_eq!("mixed content has 5 vowels", mixed_result.count, 5);

        // Unicode content
        let unicode_input = create_test_input("café naïve résumé");
        let Json(unicode_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &unicode_input)?;
        xtp_test::assert_gt!("unicode content has vowels", unicode_result.count, 0);

        Ok(())
    })?;

    // Consistency check - same input should give same result
    let consistency_input = create_test_input("test consistency");
    let Json(result1): Json<CharacterReport> = xtp_test::call("CountCharacters", &consistency_input)?;
    let Json(result2): Json<CharacterReport> = xtp_test::call("CountCharacters", &consistency_input)?;
    xtp_test::assert_eq!("consistent results", result1.count, result2.count);
    xtp_test::assert_eq!("consistent character set", &result1.characters, &result2.characters);

    // Test static_data configuration support
    xtp_test::group("static_data configuration tests", || {
        // Test custom character set
        let custom_input = create_test_input_with_config("hello123world", Some("123456789"), None);
        let Json(custom_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &custom_input)?;
        xtp_test::assert_eq!("custom digits count", custom_result.count, 3);
        xtp_test::assert_eq!("uses custom character set", &custom_result.characters, "123456789");

        // Test case sensitive mode
        let case_input = create_test_input_with_config("Hello WORLD", Some("elo"), Some(true));
        let Json(case_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &case_input)?;
        xtp_test::assert_eq!("case sensitive count", case_result.count, 4); // "e", "l", "l", "o"

        // Test case insensitive mode (explicit)
        let insensitive_input = create_test_input_with_config("Hello WORLD", Some("elo"), Some(false));
        let Json(insensitive_result): Json<CharacterReport> = xtp_test::call("CountCharacters", &insensitive_input)?;
        xtp_test::assert_eq!("case insensitive count", insensitive_result.count, 6); // "e", "l", "l", "o", "o", "l"

        Ok(())
    })?;

    Ok(())
}