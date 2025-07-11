// THIS FILE WAS GENERATED BY `xtp-rust-bindgen`. DO NOT EDIT.

#![allow(non_snake_case)]
#![allow(unused_macros)]
use extism_pdk::*;

#[allow(unused)]
fn panic_if_key_missing() -> ! {
    panic!("missing key");
}

pub(crate) mod internal {
    pub(crate) fn return_error(e: extism_pdk::Error) -> i32 {
        let err = format!("{:?}", e);
        let mem = extism_pdk::Memory::from_bytes(&err).unwrap();
        unsafe {
            extism_pdk::extism::error_set(mem.offset());
        }
        -1
    }
}

#[allow(unused)]
macro_rules! try_input {
    () => {{
        let x = extism_pdk::input();
        match x {
            Ok(x) => x,
            Err(e) => return internal::return_error(e),
        }
    }};
}

#[allow(unused)]
macro_rules! try_input_json {
    () => {{
        let x = extism_pdk::input();
        match x {
            Ok(extism_pdk::Json(x)) => x,
            Err(e) => return internal::return_error(e),
        }
    }};
}

use base64_serde::base64_serde_type;

base64_serde_type!(Base64Standard, base64::engine::general_purpose::STANDARD);

mod exports {
    use super::*;

    #[no_mangle]
    pub extern "C" fn CountCharacters() -> i32 {
        let ret = crate::count_characters(try_input!())
            .and_then(|x| extism_pdk::output(extism_pdk::Json(x)));

        match ret {
            Ok(()) => 0,
            Err(e) => internal::return_error(e),
        }
    }
}

pub mod types {
    use super::*;

    #[derive(
        Default,
        Debug,
        Clone,
        serde::Serialize,
        serde::Deserialize,
        extism_pdk::FromBytes,
        extism_pdk::ToBytes,
    )]
    #[encoding(Json)]
    pub struct CharacterReport {
        /// The count of matching characters for input string.
        #[serde(rename = "count")]
        pub count: i32,

        /// The set of characters used to get the count, e.g. "aAeEiIoOuU", "0123456789", etc.
        #[serde(rename = "characters")]
        pub characters: String,
    }
}

mod raw_imports {
    use super::*;
    #[host_fn]
    extern "ExtismHost" {}
}
