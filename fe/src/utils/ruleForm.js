export function createRuleForm(rule = {}) {
    return {
        id: rule.id || 0,
        name: rule.name || "",
        sort: rule.sort || 0,
        rules: Array.isArray(rule.rules)
            ? rule.rules.map(condition => ({...condition}))
            : [{field: "", type: "", rule: ""}],
        action: rule.action || "",
        params: rule.params || "",
    };
}
