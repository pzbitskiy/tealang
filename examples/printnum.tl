let _ = 0

let ind = 1

function printch(digit) {
  let ascii = digit + 48
  let dbg = apps[0].get("dbg")
  apps[0].put("dbg",setbyte(apps[0].get("dbg"), ind, ascii))
  ind = ind + 1
  return 1
}

function printFirstDigit(nn) {
    if nn >= 100 {
         _= printch(nn / 100)
         return nn % 100
    }
    if nn >= 10 {
            _= printch(nn / 10)
            return nn % 10
    }
    _= printch(nn)
    return 9999
}

function printNum(num) {
    log("Start of printNum.")
    let left = printFirstDigit(num)
    if left != 9999 {
        left = printFirstDigit(left)
    }
    if left != 9999 {
        left = printFirstDigit(left)
    }
    return 1
}

function logic() {
    apps[0].put("dbg", "            ")
    _= printNum(74)

    return 1
}

