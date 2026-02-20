#!/bin/bash
# Fix all CostAnalysis response parsing
sed -i.bak3 '
234,236 {
    /var response CostAnalysis/ {
        i\			var apiResp APIResponse
        c\			json.NewDecoder(w.Body).Decode(&apiResp)\
\
			// Convert apiResp.Data to CostAnalysis\
			dataBytes, _ := json.Marshal(apiResp.Data)\
			var response CostAnalysis\
			json.Unmarshal(dataBytes, &response)
        n
        d
    }
}

272,274 {
    /var response CostAnalysis/ {
        i\	var apiResp APIResponse
        c\	json.NewDecoder(w.Body).Decode(&apiResp)\
\
	// Convert apiResp.Data to CostAnalysis\
	dataBytes, _ := json.Marshal(apiResp.Data)\
	var response CostAnalysis\
	json.Unmarshal(dataBytes, &response)
        n
        d
    }
}

310,312 {
    /var response CostAnalysis/ {
        i\	var apiResp APIResponse
        c\	json.NewDecoder(w.Body).Decode(&apiResp)\
\
	// Convert apiResp.Data to CostAnalysis\
	dataBytes, _ := json.Marshal(apiResp.Data)\
	var response CostAnalysis\
	json.Unmarshal(dataBytes, &response)
        n
        d
    }
}

409,411 {
    /var response CostAnalysis/ {
        i\	var apiResp APIResponse
        c\	json.NewDecoder(w.Body).Decode(&apiResp)\
\
	// Convert apiResp.Data to CostAnalysis\
	dataBytes, _ := json.Marshal(apiResp.Data)\
	var response CostAnalysis\
	json.Unmarshal(dataBytes, &response)
        n
        d
    }
}

474,476 {
    /var response CostAnalysis/ {
        i\			var apiResp APIResponse
        c\			json.NewDecoder(w.Body).Decode(&apiResp)\
\
			// Convert apiResp.Data to CostAnalysis\
			dataBytes, _ := json.Marshal(apiResp.Data)\
			var response CostAnalysis\
			json.Unmarshal(dataBytes, &response)
        n
        d
    }
}
' handler_costing_test.go
