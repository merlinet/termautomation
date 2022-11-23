package constdef

import (
	"time"
)

var PRODUCT_NAME string = "termautomation"
var PRODUCT_VERSION string = "1.0.0"

var RECORD_VERSION float64 = 3

/* recorder interval 모드
 * expect: input prompt 가 나오면 send, expect record 기록
 * interact: key input interval을 기록, send, sleep record 기록
 */
var MODE_EXPECT uint8 = 0x01
var MODE_INTERACT uint8 = 0x02

/* terminal write(send) 시 sleep time 조정
 */
var SEND_INTERVAL_MILLISECOND time.Duration = 50
var OUTPUT_TIMEOUT_MILLISECOND time.Duration = 50 // OUTPUT Timeout

/* recorder filter 분기 구분
 */
var IO_SELECTER_INPUT uint8 = 0x01
var IO_SELECTER_OUTPUT uint8 = 0x02
var IO_SELECTER_TIMEOUT uint8 = 0x03

var CMD_CONTROL_MODE_BLOCK uint8 = 0x01
var CMD_CONTROL_MODE_AUTO uint8 = 0x02
var CMD_CONTROL_MODE_IGNORE_RECORD uint8 = 0x03

/* test 결과 코드
 */
var SUCCESS uint8 = 0x00
var FAIL uint8 = 0x01
var NA uint8 = 0x02
var NOP uint8 = 0x03

/* record version 2 호환
 */
var CHECKER_SUCCESS uint8 = 0x00
var CHECKER_FAIL uint8 = 0x01
var CHECKER_NA uint8 = 0x02
var CHECKER_NOP uint8 = 0x03

const (
	RCMD_CHECK_NOP               = 0x00
	RCMD_CHECK_EXIT_CODE         = 0x01
	RCMD_CHECK_OUTPUT_STRING     = 0x02
	RCMD_CHECK_OUTPUT_LINE_COUNT = 0x03

	RCMD_CHECK_NOP_STR               = "nop"
	RCMD_CHECK_EXIT_CODE_STR         = "exit_code"
	RCMD_CHECK_OUTPUT_STRING_STR     = "output_string"
	RCMD_CHECK_OUTPUT_LINE_COUNT_STR = "output_line_count"
)

var ANSI_WHITE string = "\033[0;37m"
var ANSI_GRAY string = "\033[0;30m"
var ANSI_GRAY_BOLD string = "\033[7;30m"
var ANSI_RED_BOLD string = "\033[7;31m"
var ANSI_GREEN string = "\033[0;32m"
var ANSI_GREEN_BOLD string = "\033[7;32m"
var ANSI_YELLOW2 string = "\033[7;33m"
var ANSI_YELLOW string = "\033[0;33m"
var ANSI_YELLOW_BOLD string = "\033[1;33m"
var ANSI_BLUE_BOLD string = "\033[1;34m"
var ANSI_CYAN string = "\033[0;36m"
var ANSI_CYAN_BOLD string = "\033[1;36m"
var ANSI_END string = "\033[0m"

var INSTALL_ROOT_CONF string = "/etc/discovery.conf"
var TMP_DIR string = "/tmp/discovery"

var DEBUG_MODE_BASH_PROMPT string = "bash-"

var DEFAULT_EXPECT_TIMEOUT float64 = 10       // 10초
var LOGIN_EXPECT_TIMEOUT float64 = 120        // 120초, rss 패킷 생성 worker의 경우 로드가 많아 느림
var DEFAULT_EXPECT_TIMEOUT_STEP float64 = 250 // 250ms

var DEFAULT_CHARACTER_SET string = "utf8"

var DEFAULT_PROMPT_RE_STR string = `^.*[#\$>:]\s*$`
var BASH_PROMPT_RE_STR string = `^.*[#\$>]\ $`
var BASH_PROMPT2_RE_STR string = `^.*[#\$]\ $`

var EOL_LF string = "lf"     // \n
var EOL_CR string = "cr"     // \r
var EOL_CRLF string = "crlf" // \r\n
var DEFAULT_EOL string = EOL_LF

var MAX_OUTPUT_LINE_COUNT uint32 = 10000
var MAX_RECORDER_OUTPUT_LINE_COUNT uint32 = 500

/* output string, exit code 의 default 변수 이름
 */
var OUTPUT_STRING_VARIABLE_NAME string = "output_string"
var EXIT_CODE_VARIABLE_NAME string = "exit_code"

/* rest response 변수 suffix
 */
var REST_RESPONSE_IS_SUCCESS_VARIABLE_NAME_SUFFIX string = "_success"
var REST_RESPONSE_STATUS_CODE_VARIABLE_NAME_SUFFIX string = "_status_code"
var REST_RESPONSE_BODY_VARIABLE_NAME_SUFFIX string = "_response"

var ARGS_VARIABLE_NAME string = "args"

/* debug flag
 */
var DEBUG bool = true
