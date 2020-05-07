import stdlib.templates
import mymodule

function logic() {
    let to = addr"PNWOET7LLOWMBMLE4KOCELCX6X3D3Q4H2Q4QJASYIEOF7YIPPQBG3YQ5YI"
    let amount = 2000
    let result = DynamicFee(to, amount, global.ZeroAddress, 1, 1000, getLease())
    return result
}
