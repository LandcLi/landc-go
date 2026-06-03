package cmd

import (
	"strings"
)

// ParserOption 解析选项
type ParserOption struct {
	CaseSensitive bool // 是否大小写敏感
	Strict        bool // 是否严格模式，遇到无效选项时返回错误
}

// Parser 命令行参数解析器
type Parser struct {
	args             []string
	options          map[string]string
	arguments        []string
	supportedOptions map[string]bool   // 支持的选项，键为选项名，值为是否需要参数
	optionAliases    map[string]string // 选项别名映射，键为别名，值为主要选项名
	parserOptions    ParserOption      // 解析选项
}

// NewParser 创建一个新的参数解析器
func NewParser(args []string) *Parser {
	p := &Parser{
		args:             args,
		options:          make(map[string]string),
		arguments:        []string{},
		supportedOptions: make(map[string]bool),
		optionAliases:    make(map[string]string),
	}
	p.parse()
	return p
}

// Parse 创建一个新的参数解析器，支持配置选项
func Parse(supportedOptions map[string]bool, option ...ParserOption) (*Parser, error) {
	return ParseWithArgs(nil, supportedOptions, option...)
}

// ParseWithArgs 使用指定的参数创建一个新的参数解析器
func ParseWithArgs(args []string, supportedOptions map[string]bool, option ...ParserOption) (*Parser, error) {
	if args == nil {
		args = []string{}
	}

	p := &Parser{
		args:             args,
		options:          make(map[string]string),
		arguments:        []string{},
		supportedOptions: make(map[string]bool),
		optionAliases:    make(map[string]string),
	}

	// 处理解析选项
	if len(option) > 0 {
		p.parserOptions = option[0]
	}

	// 处理支持的选项，包括别名
	for optStr, hasValue := range supportedOptions {
		optNames := strings.Split(optStr, ",")
		if len(optNames) == 0 {
			continue
		}

		// 第一个作为主要选项名
		mainOpt := optNames[0]
		p.supportedOptions[mainOpt] = hasValue

		// 其余作为别名
		for i := 1; i < len(optNames); i++ {
			alias := optNames[i]
			p.optionAliases[alias] = mainOpt
			p.supportedOptions[alias] = hasValue
		}
	}

	// 解析参数
	err := p.parse()
	return p, err
}

// parse 解析命令行参数
func (p *Parser) parse() error {
	if len(p.args) < 1 {
		return nil
	}

	for i := 1; i < len(p.args); i++ {
		arg := p.args[i]

		// 处理选项
		if strings.HasPrefix(arg, "-") {
			// 处理长选项 --option=value
			if strings.HasPrefix(arg, "--") {
				parts := strings.SplitN(arg[2:], "=", 2)
				optName := parts[0]

				// 检查选项是否支持
				if len(p.supportedOptions) > 0 {
					_, ok := p.supportedOptions[optName]
					if !ok {
						// 检查是否是别名
						_, isAlias := p.optionAliases[optName]
						if !isAlias {
							if p.parserOptions.Strict {
								return nil // 严格模式下遇到未知选项返回错误
							}
						}
					}
				}

				if len(parts) == 2 {
					// 处理 --option=value 形式
					p.setOptionValue(optName, parts[1])
				} else {
					// 处理 --option 形式
					hasValue := false
					if val, ok := p.supportedOptions[optName]; ok {
						hasValue = val
					}

					if hasValue && i+1 < len(p.args) && !strings.HasPrefix(p.args[i+1], "-") {
						// 选项需要参数，且下一个参数不是选项
						p.setOptionValue(optName, p.args[i+1])
						i++
					} else {
						// 选项不需要参数或下一个参数是选项
						p.setOptionValue(optName, "")
					}
				}
			} else {
				// 处理短选项 -o value
				if len(arg) == 2 {
					optName := string(arg[1])

					// 检查选项是否支持
					if len(p.supportedOptions) > 0 {
						_, ok := p.supportedOptions[optName]
						if !ok {
							// 检查是否是别名
							_, isAlias := p.optionAliases[optName]
							if !isAlias {
								if p.parserOptions.Strict {
									return nil // 严格模式下遇到未知选项返回错误
								}
							}
						}
					}

					// 检查选项是否需要参数
					hasValue := false
					if val, ok := p.supportedOptions[optName]; ok {
						hasValue = val
					} else if len(p.supportedOptions) == 0 {
						// 如果没有配置支持的选项，默认短选项需要参数
						hasValue = true
					}

					if hasValue && i+1 < len(p.args) && !strings.HasPrefix(p.args[i+1], "-") {
						// 选项需要参数，且下一个参数不是选项
						p.setOptionValue(optName, p.args[i+1])
						i++
					} else {
						// 选项不需要参数或下一个参数是选项
						p.setOptionValue(optName, "")
					}
				}
			}
		} else {
			// 处理普通参数
			p.arguments = append(p.arguments, arg)
		}
	}

	return nil
}

// setOptionValue 设置选项值，处理别名
func (p *Parser) setOptionValue(optName, value string) {
	// 检查是否是别名
	if mainOpt, isAlias := p.optionAliases[optName]; isAlias {
		p.options[mainOpt] = value
		// 同时设置别名的值，方便通过别名获取
		p.options[optName] = value
	} else {
		p.options[optName] = value
		// 同时设置所有别名的值
		for alias, main := range p.optionAliases {
			if main == optName {
				p.options[alias] = value
			}
		}
	}
}

// GetArg 获取指定位置的参数
func (p *Parser) GetArg(index int) string {
	if index >= 0 && index < len(p.arguments) {
		return p.arguments[index]
	}
	return ""
}

// GetArgAll 获取所有参数
func (p *Parser) GetArgAll() []string {
	return p.arguments
}

// GetOpt 获取指定选项的值
func (p *Parser) GetOpt(name string) string {
	if val, ok := p.options[name]; ok {
		return val
	}
	return ""
}

// GetOptAll 获取所有选项
func (p *Parser) GetOptAll() map[string]string {
	return p.options
}

// GetArgs 获取所有参数
func (p *Parser) GetArgs() []string {
	return p.arguments
}

// HasOpt 检查是否存在指定选项
func (p *Parser) HasOpt(name string) bool {
	_, ok := p.options[name]
	return ok
}
