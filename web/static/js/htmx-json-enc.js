(function() {
    let api = {}
    htmx.defineExtension("json-enc", {
        init: function(hApi) {
            api = hApi
        },
        onEvent: function(name, evt) {
            if (name === "htmx:configRequest") {
                evt.detail.headers["Content-Type"] = "application/json"
            }
        },
        encodeParameters: function(_, parameters, elt) {
            let resObj = {}
            for (let k in parameters) {
                let parValue = parameters[k]
                let inpEl = elt.elements[k]

                if (!inpEl) {
                    resObj[k] = parValue
                    continue
                }

                switch (inpEl.type) {
                    case "number":
                        resObj[k] = Number(parValue)
                        break;
                    case "text":
                    default:
                        resObj[k] = parValue
                }
            }

            return JSON.stringify(resObj)
        }
    })
})()
