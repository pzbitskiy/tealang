const a = 1
const b = "string"
const c = "\x32\x33\x34";

function compare_increment(a, b) {
    let out = if a == b { a+1 } else { a }
    return out
}

function logic(txn, gtxn, args) {
    let result = compare_increment(1, 2)
    if result == 0 {
        error
    } else {
        return result
    }
}
