import { isDate, zipObject, padStart, findIndex } from 'lodash';
import { today } from '../date'
import __ from './locale'

/**
 * 格式化日期
 * @param time 时间戳或日期对象
 * @param format 格式
 */
export function formatTime(time?: number | Date, format: string = 'yyyy/MM/dd HH:mm:ss'): string {
    if (!arguments.length) {
        return '';
    }

    let d;
    if (/U.*Z$/.test(format)) {
        /**
         * UTC时间
         * 动态获取当前时区
         */
        const timeOffset = new Date().getTimezoneOffset();
        const timezone = - timeOffset / 60;
        const intervalTime = timezone * 60 * 60 * 1000;
        d = isDate(time) ? new Date(Math.round(time.getTime() - intervalTime)) : new Date(Math.round(time - intervalTime));
    } else {
        d = isDate(time) ? time : new Date(time);
    }

    let year = d.getFullYear();
    let month = padStart(String(d.getMonth() + 1), 2, '0');
    let date = padStart(String(d.getDate()), 2, '0');
    let hour = padStart(String(d.getHours()), 2, '0');
    let minute = padStart(String(d.getMinutes()), 2, '0');
    let second = padStart(String(d.getSeconds()), 2, '0');

    const handledTime = format.replace('U', ' U ').replace('Z', ' Z').replace(/\b(\w+)\b/g, function (match) {
        switch (match) {
            case 'yyyy':
                return year;

            case 'MM':
                return month;

            case 'dd':
                return date;

            case 'U':
                return 'T';

            case 'HH':
                return hour;

            case 'mm':
                return minute;

            case 'ss':
                return second;

            case 'Z':
                return 'Z'
        }
    });

    return handledTime.replace(' T ', 'T').replace(' Z', 'Z')
}

/**
 * 格式化时分秒
 * @param time 时间 秒
 */
export function secToHHmmss(time) {
    return `${padStart(Math.floor(time / 3600), 2, '0')}:${padStart(Math.floor(time % 3600 / 60), 2, '0')}:${padStart(Math.round(time % 60), 2, '0')}`
}

/**
 * 转换字节数
 * @param bytes 字节大小
 * @param units 单位集合
 * @param minUnit 最小单位
 * @return size 大小 unit单位
 */
// eslint-disable-next-line
export function transformBytes(bytes: number, { minUnit = 'B' } = {} as { minUnit: string }): [number, string] {
    // 单位集合
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB', 'BB', 'NB', 'DB'];
    // 最小显示单位
    let minUnitIndex = findIndex(units, (value) => value === minUnit);
    // 下标，用来计算适合单位的下标
    let index;
    for (index = minUnitIndex; index <= units.length; index++) {
        if (index === units.length - 1 || bytes < Math.pow(1024, index + 1)) {
            break;
        }
    }
    return [bytes / Math.pow(1024, index), units[index]];
}

/**
 * 大小格式化
 * @param bytes 字节大小
 * @param fixed 保留位数 minUnit 最小显示单位
 * @return 返回格式化后的大小字符串
 */
// eslint-disable-next-line
export function formatSize(bytes: number, fixed: number = 2, { minUnit = 'B' } = {} as { minUnit: string }): string {
    if (bytes === undefined) {
        return ''
    }

    const [size, unit] = transformBytes(bytes, { minUnit })

    if (bytes === size) {
        return size + unit;
    } else {
        const sizeStr = size.toString()

        // 不能使用toFixed(fixed)，会导致类似4.99998被入为5.00
        if (sizeStr.indexOf('.') === -1) {
            return sizeStr + unit
        } else {
            const indexOfPoint = sizeStr.indexOf('.');
            return sizeStr.slice(0, indexOfPoint + fixed + 1) + unit;
        }
    }
}

/**
 * 速率格式化
 * @param bytes 字节大小
 * @param fixed 保留位数 minUnit 最小显示单位
 * @return 返回格式化后的大小字符串
 */
// eslint-disable-next-line
export function formatRate(bytes: number, fixed: number = 2, { minUnit = 'B' } = {} as { minUnit: string }): string {
    if (bytes === undefined) {
        return ''
    }
    const [size, unit] = transformBytes(bytes, { minUnit });
    return size.toFixed(fixed) + unit + '/s';
}

/**
 * 根据字符串模板从字符串中提取键值对
 * @param input 要匹配的文本
 * @param template 匹配模板
 * @returns 返回匹配到的键值对
 */
export function matchTemplate(input: string, template: string): Record<string, any> {
    let names: ReadonlyArray<string> = [];
    let regExpStr = template.replace(/\${\s*(\w+?)\s*}/g, function () {
        names = [...names, arguments[1]]
        return '(.+)';
    });
    let pattern = new RegExp(regExpStr);
    let result = pattern.exec(input);
    let values = result.slice(1);

    return zipObject(names, values);
}

/**
 * 裁切字符串长度
 * @param input string 输入字符串
 * @param [options] {object} 裁切选项
 * @param [options.limit = 20] {number} 限制字符长度
 * @param [options.indicator = '...'] {string} 截取表示字符串
 * @return string
 */
export function shrinkText(input: string = '', { limit = 20, indicator = '...' } = {}): string {
    const CHS_CHAR_REG = /[\u0391-\uFFE5]/g;
    const indicatorWidth = indicator.length + (indicator.match(CHS_CHAR_REG) || []).length; // 每个中文字符记数+1
    const allowStringWidth = limit - indicatorWidth; // 允许的字符宽度
    let rawCut = String(input).slice(0, allowStringWidth); // 先进行无差别切片, 包含中文和英文
    let rawCutChsCount = rawCut.match(CHS_CHAR_REG);
    let inputCutCount = input.match(CHS_CHAR_REG) ? input.match(CHS_CHAR_REG).length : 0

    if ((input.length + inputCutCount) <= limit) {
        return input;
    } else {
        // 当最近非ASCII字符在限制长度之外
        if (!rawCutChsCount) {
            return rawCut + indicator;
        }
        // 当非ASCII字符在限制长度内
        else {
            const chars = rawCut.split('');
            let charCount = 0;
            let i = 0; // 切片尾指针

            for (let len = chars.length; i < len; i++) {
                charCount += chars[i].match(CHS_CHAR_REG) ? 2 : 1;

                if (charCount <= allowStringWidth) {
                    continue;
                } else {
                    break;
                }
            }

            return rawCut.slice(0, i) + indicator;
        }
    }

}

/**
 * 格式化颜色，色值加 #
 */
export function formatColor(input) {
    return /^#/.test(String(input)) ? input : `#${input}`
}

/**
 * 裁剪文件名(省略中间的字符串)，除英文字符外的字符认为长度为2
 * @param name 要裁剪的字符串
 * @param param1 最大长度，默认70
 */
export function decorateText(name, { limit = 70 }) {
    function sumLens(str) {
        let charCode = -1, realLength = 0;
        for (let i = 0; i < str.length; i++) {
            charCode = str.charCodeAt(i);
            if (charCode >= 0 && charCode <= 128) {
                realLength += 1;
            } else {
                realLength += 2;
            }
        }
        return realLength;
    }

    if (!name) {
        return '';
    }
    let realLength = sumLens(name)
    let len = name.length, charCode = -1;

    if (realLength > limit) {
        let reset = Math.floor(limit / 2);
        let tmpIndex = 0;
        let tmpLens = 0;
        let resLeftStr = '';
        let resRightStr = '';
        while (tmpLens < reset) {
            charCode = name.charCodeAt(tmpIndex);
            (charCode >= 0 && charCode <= 128) ? tmpLens += 1 : tmpLens += 2;
            resLeftStr += name[tmpIndex];
            tmpIndex += 1;

        }
        tmpIndex = len - 1;
        tmpLens = 0;
        while (tmpLens < reset) {
            charCode = name.charCodeAt(tmpIndex);
            (charCode >= 0 && charCode <= 128) ? tmpLens += 1 : tmpLens += 2;
            resRightStr += name[tmpIndex];
            tmpIndex -= 1;

        }

        if (sumLens(resLeftStr) + sumLens(resRightStr) === sumLens(name)) {
            return name
        } else {
            return resLeftStr + '...' + resRightStr.split('').reverse().join('');
        }

    }
    return name;

}

/**
 * 格式化日期
 * @param {number} modified-后台传递的原始时间戳/1000
 * @param {string} timeFormat-日期时间格式
 * @returns {string} 按照今天，昨天和其他时间的方式显示
 */
export function formatTimeRelative(modified: number, timeFormat: string = 'HH:mm:ss'): string {
    const startOfToday: number = (new Date(today().getFullYear(), today().getMonth(), today().getDate(), 0, 0, 0, 0)).getTime() // 获取今天开始时间的时间戳 00:00:00

    const endOfToday: number = startOfToday + (24 * 3600 * 1000 - 1); // 今天结束时间的时间戳 23:59:59

    const startOfYesterday: number = startOfToday - (24 * 3600 * 1000); // 昨天开始时间的时间戳 00:00:00

    const endOfYesterday: number = startOfToday - 1; // 昨天结束时间的时间戳 23:59:59

    if (modified >= startOfToday && modified <= endOfToday) {
        // 处理 timeFormat 为空字符串的情况
        return `${__('今天')}${timeFormat && (' ' + formatTime(modified, timeFormat)) || ''}`
    } else if (modified >= startOfYesterday && modified <= endOfYesterday) {
        return `${__('昨天')}${timeFormat && (' ' + formatTime(modified, timeFormat)) || ''}`
    } else {
        return formatTime(modified, `yyyy/MM/dd${timeFormat ? ` ${timeFormat}` : ''}`)
    }
}

/**
 * 隐藏一部分手机号
 * @param {string} phoneNumber-传递的手机号
 * @returns {string} 显示如：152*****146
 */
export function maskPhoneNumber(phoneNumber: string): string {
    return phoneNumber.replace(/(\d{3})(\d{5})(\d{3})/g, '$1*****$3');
}

/**
 * 隐藏一部分邮箱地址
 * @param {string} email-传递的邮箱地址
 * @returns {string} 显示如：729*****58@qq.com
 */
export function maskEmail(email: string): string {
    return email.replace(/(.{3}).+(.{2}@.+)/g, '$1*****$2');
}

/**
 * 容量单位转换
 * @param {number | string | undefined} input  输入值
 * @param {string} from  输入单位
 * @param {string} to  输出单位
 * @returns {number|undefined}
 */
export function convertUnit(input: number | string, from: string, to: string): number | undefined {
    if (input === '' || input === undefined) {
        return undefined;
    }

    const figure = Number(input),
        inUnit = from.slice(0, 1).toUpperCase(), // 对*B单位做兼容处理
        outUnit = to.slice(0, 1).toUpperCase(); // 对*B单位做兼容处理

    const unitLevel = {
        B: 0,
        K: 1,
        M: 2,
        G: 3,
        T: 4,
        P: 5,
    };

    const exponent = Math.pow(1024, (unitLevel[inUnit] - unitLevel[outUnit]));

    return figure * exponent;
}

/**
 * 对number获取lower和upper之间的数字
 * @param {number} num 要比较的数字
 * @param {number} lower 下限
 * @param {number} upper 上限
 */
export function clamp(num: number, lower: number, upper: number): number {
    num = +num
    lower = +lower
    upper = +upper
    lower = lower === lower ? lower : 0
    upper = upper === upper ? upper : 0
    if (num === num) {
        num = num <= upper ? num : upper
        num = num >= lower ? num : lower
    }
    return num
}

/**
 * 将下划线转小驼峰
 * @param 下划线字符串
 * @result 转换为小驼峰的字符串
 */
export function toCamel(param: string): string {
    return param.replace(/\_(\w)/g, function (all, letter) {
        return letter.toUpperCase();
    });
}

/**
 * 将小驼峰转下划线
 * @param 小驼峰字符串
 * @result 转换为下划线的字符串
 */
export function toUnderline(param: string): string {
    return param.replace(/([A-Z])/g, '_$1').toLowerCase();
}