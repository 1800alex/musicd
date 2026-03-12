/* eslint-disable @typescript-eslint/no-unsafe-function-type */

import path from "path";
import * as _ from "lodash-es";

/* Additional */
const closeTo = function (actual: number, expected: number, delta: number) {
	return _.inRange(actual, expected - delta, expected + delta);
};

const max = function (a: number | number[], b?: number) {
	if (Array.isArray(a) && b === undefined) {
		return _.max(a);
	}

	if (!Array.isArray(a) && b !== undefined) {
		return Math.max(a, b);
	}

	throw new Error("Must specify an array or two numbers");
};

const min = function (a: number | number[], b?: number) {
	if (Array.isArray(a) && b === undefined) {
		return _.min(a);
	}

	if (!Array.isArray(a) && b !== undefined) {
		return Math.min(a, b);
	}

	throw new Error("Must specify an array or two numbers");
};

/* Strings */
// todo should this return null or index instead of -1 or index like in C?
const strstr = function (haystack: string, needle: string) {
	return haystack.indexOf(needle);
};

// todo should this return length of a or 0 instead of true/false like in C?
const strcmp = function (a: string, b: string) {
	return a === b;
};

const Uint8ArrayStringify = function (b: Uint8Array) {
	let result = "";
	const len = b.length;
	for (let i = 0; i < len; i++) {
		if (0 !== i) {
			result += " ";
		}

		result += `0${Number(b[i]).toString(16)}`.slice(-2).toUpperCase();
	}

	return result;
};

const Int8ArrayStringify = Uint8ArrayStringify;

const Uint16ArrayStringify = function (b: Uint16Array) {
	let result = "";
	const len = b.length;
	for (let i = 0; i < len; i++) {
		if (0 !== i) {
			result += " ";
		}

		result += `000${Number(b[i]).toString(16)}`.slice(-4).toUpperCase();
	}

	return result;
};

const Int16ArrayStringify = Uint16ArrayStringify;

const Uint32ArrayStringify = function (b: Uint32Array) {
	let result = "";
	const len = b.length;
	for (let i = 0; i < len; i++) {
		if (0 !== i) {
			result += " ";
		}

		result += `0000000${Number(b[i]).toString(16)}`.slice(-8).toUpperCase();
	}

	return result;
};

const Int32ArrayStringify = Uint32ArrayStringify;

const filename = function (a: string) {
	const b = a || "";

	const result = {
		Filename: "",
		FileExtension: "",
		Basename: path.basename(b),
		Dirname: path.dirname(b),
		Path: path.resolve(b),
		PathDirname: path.dirname(path.resolve(b))
	};

	const lastIndex = result.Basename.lastIndexOf(".");
	const nameLength = result.Basename.length;

	if (lastIndex > 0) {
		if (lastIndex === nameLength - 1) {
			/* String ends with a period */
			result.Filename = result.Basename;
		} else {
			// console.log(last_index);
			result.Filename = result.Basename.substr(0, lastIndex) || result.Basename;
			result.FileExtension = result.Basename.substr(lastIndex + 1, nameLength - 1) || result.Basename;
		}
	} else {
		/* No period found */
		result.Filename = result.Basename;
	}

	return result;
};

// TODO: Should use DotProp for increased code reusability.
const getNested = function (obj: any, dotSeparatedKeys: string, defaultValue?: any) {
	let fallback;

	if (arguments.length > 2) {
		fallback = defaultValue;
	}

	if (arguments.length > 1 && typeof dotSeparatedKeys !== "string") {
		return fallback;
	}

	if (typeof obj !== "undefined" && "string" === typeof dotSeparatedKeys) {
		const pathArr = dotSeparatedKeys.split(".");
		pathArr.forEach((key, idx, arr) => {
			if ("string" === typeof key && key.includes("[")) {
				try {
					// extract the array index as string
					const match = /\[([^)]+)\]/.exec(key);
					if (_.isNil(match)) {
						throw new Error("Did not match");
					}

					const pos = match[1];
					// get the index string length (i.e. '21'.length === 2)
					const posLen = pos.length;
					arr.splice(idx + 1, 0, `${Number(pos)}`);

					// keep the key (array name) without the index comprehension:
					// (i.e. key without [] (string of length 2)
					// and the length of the index (posLen))
					arr[idx] = key.slice(0, -2 - posLen);
				} catch (e) {
					// do nothing
				}
			}
		});

		obj = pathArr.reduce((o, key) => (o && o[key] !== "undefined" ? o[key] : undefined), obj);

		if ("undefined" === typeof obj) {
			return fallback;
		}
	}

	return obj;
};

const defaultsToOneOf = (...list: any[]) => {
	//const args = Array.prototpye.concat.call(arguments);
	for (let i = 0; i < list.length; i++) {
		if (i + 1 >= list.length) {
			return list[i];
		}
		list[i + 1] = _.defaultTo(list[i], list[i + 1]);
	}

	return undefined;
};

// NOTE: This is bad design in most cases... You should be using Promise.all,
// but this is not always possible.
const asyncEach = async (list: any[], callback: Function) => {
	for (let i = 0; i < list.length; i++) {
		await callback(list[i], i);
	}
};

async function asyncForEach<T, R>(array: T[], callback: (element: T) => Promise<R>): Promise<R[]> {
	const promises: Promise<R>[] = array.map(callback);
	return Promise.all(promises);
}

// https://github.com/lodash/lodash/issues/1244
const mapKeysDeep = (obj: any, cb: Function): any =>
	_.mapValues(_.mapKeys(obj, cb), (val) => (_.isObject(val) ? mapKeysDeep(val, cb) : val));

const mapValuesDeep = (val: any, cb: Function): any =>
	_.isObject(val) ? _.mapValues(val, (val_2) => mapValuesDeep(val_2, cb)) : cb(val);

const errToString = (err: any) => {
	if (err instanceof Error) {
		if (err.message) {
			return `: ${err.message}`;
		}

		return `: ${err}`;
	}
	if ("message" in err) {
		return `: ${err.message}`;
	}
	if ("string" === typeof err) {
		return `: ${err}`;
	}

	return "";
};

const setParameter = (object: any, key: string, value: any) => {
	if ("" === key || "." === key) {
		if (_.isString(value) || _.isNumber(value) || _.isBoolean(value) || _.isNil(value)) {
			object = value;
		}

		// It's an object so we must copy the properties over
		else if (_.isObject(value)) {
			// Clear out existing keys in object
			Object.keys(object).forEach((k) => {
				delete (object as any)[k];
			});

			// Deep clone src into object
			Object.assign(object, _.cloneDeep(value));
			return object;
		}

		return object;
	}

	return _.set(object, key, value);
};

const getParameter = (object: any, key: string) => {
	if ("" === key || "." === key) {
		return object;
	}

	return _.get(object, key);
};

const getParameterCaseInsensitive = (object: any, key: string) => {
	if ("" === key || "." === key) {
		return object;
	}

	const asLowercase = key.toLowerCase();
	const found = Object.keys(object).find((k) => k.toLowerCase() === asLowercase);
	if (!_.isNil(found)) {
		return object[found];
	}

	return _.get(object, key);
};

const isFiniteNumber = (value: unknown): value is number => _.isNumber(value) && _.isFinite(value);

// NOTE: If you're using this, you're probably better off just using _.isError()
// https://lodash.com/docs/4.17.15#isError
const isErrnoException = (error: unknown): error is NodeJS.ErrnoException => {
	if ("object" === typeof error && error !== null) {
		const exception = <NodeJS.ErrnoException>error;
		if (
			("number" === typeof exception.errno || "undefined" === typeof exception.errno) &&
			("string" === typeof exception.code || "undefined" === typeof exception.code) &&
			("string" === typeof exception.path || "undefined" === typeof exception.path) &&
			("string" === typeof exception.syscall || "undefined" === typeof exception.syscall)
		) {
			return true;
		}
	}

	return false;
};

const MACAddress = {
	toString: (macAddress: string): string => MACAddress.fromUint8Array(MACAddress.toUint8Array(macAddress)),
	toStringWithoutColons: (macAddress: string): string =>
		MACAddress.fromUint8ArrayWithoutColons(MACAddress.toUint8Array(macAddress)),
	toUint8Array: (macAddress: string): Uint8Array => {
		// Remove all non-hexadecimal characters from the input string
		const cleanedMAC = macAddress.replace(/[^A-Fa-f0-9]/g, "");

		// Create an array to hold the bytes of the MAC address
		const macBytes: number[] = [];

		// Use regular expression to match the groups of two characters and parse them as integers
		const matches = cleanedMAC.match(/.{2}/g);

		if (matches && 6 === matches.length) {
			_.each(matches, (match) => {
				const byte = parseInt(match, 16);
				if (_.isNaN(byte) || byte < 0 || byte > 255) {
					throw new Error(`Invalid MAC address: ${macAddress}`);
				}
				macBytes.push(byte);
			});
		} else {
			throw new Error("Invalid MAC address format");
		}

		// Create a Uint8Array from the macBytes array
		return new Uint8Array(macBytes);
	},
	toBigInt: (macAddress: string): bigint => {
		const bytes = MACAddress.toUint8Array(macAddress);
		const buf = Buffer.from([0, 0, bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]]);

		return buf.readBigUInt64BE(0);
	},
	fromBigInt: (macAddress: bigint): string => {
		const buf = Buffer.alloc(8);
		buf.writeBigUInt64BE(macAddress, 0);

		return MACAddress.fromUint8Array(new Uint8Array([buf[2], buf[3], buf[4], buf[5], buf[6], buf[7]]));
	},
	fromUint8Array: (macAddress: Uint8Array): string => {
		if (macAddress.length !== 6) {
			throw new Error(`Invalid MAC address: ${macAddress}`);
		}

		return Array.from(macAddress, (byte) => byte.toString(16).padStart(2, "0"))
			.join(":")
			.toUpperCase();
	},
	fromUint8ArrayWithoutColons: (macAddress: Uint8Array): string => {
		if (macAddress.length !== 6) {
			throw new Error(`Invalid MAC address: ${macAddress}`);
		}

		return Array.from(macAddress, (byte) => byte.toString(16).padStart(2, "0"))
			.join("")
			.toUpperCase();
	}
};

const minOfEnum = (e: object) => {
	const values = Object.keys(e)
		.map((k) => ("" === k ? NaN : Number(k)))
		.filter((k) => !isNaN(k));
	return Math.min(...values);
};

const maxOfEnum = (e: object) => {
	const values = Object.keys(e)
		.map((k) => ("" === k ? NaN : Number(k)))
		.filter((k) => !isNaN(k));
	return Math.max(...values);
};

const shuffle = <Type>(arr: Type[]): Type[] => {
	let rand: number;
	let temp: Type;
	let i: number;

	for (i = arr.length - 1; i > 0; i -= 1) {
		rand = Math.floor((i + 1) * Math.random()); //get random between zero and i (inclusive)
		temp = arr[rand];
		arr[rand] = arr[i]; //swap i (last element) with random element.
		arr[i] = temp;
	}
	return arr;
};

const isArrayEqual = <Type>(a: Type[], b: Type[], compare: (a: Type, b: Type) => boolean): boolean => {
	if (a.length !== b.length) {
		return false;
	}

	for (let i = 0; i < a.length; i++) {
		if (!compare(a[i], b[i])) {
			return false;
		}
	}

	return true;
};

// Types taken from lodash
type NotVoid = unknown;
type PartialShallow<T> = {
	[P in keyof T]?: T[P] extends object ? object : T[P];
};
type List<T> = ArrayLike<T>;
type PropertyName = string | number | symbol;
type MemoObjectIterator<T, TResult, TList> = (prev: TResult, curr: T, key: string, list: TList) => TResult;
type ObjectIterator<TObject, TResult> = (value: TObject[keyof TObject], key: string, collection: TObject) => TResult;
type ObjectIteratee<TObject> = ObjectIterator<TObject, NotVoid> | IterateeShorthand<TObject[keyof TObject]>;
type ObjectIterateeCustom<TObject, TResult> = ObjectIterator<TObject, TResult> | IterateeShorthand<TObject[keyof TObject]>;
type IterateeShorthand<T> = PropertyName | [PropertyName, any] | PartialShallow<T>;
type ArrayIterator<T, TResult> = (value: T, index: number, collection: T[]) => TResult;
type ListIterator<T, TResult> = (value: T, index: number, collection: List<T>) => TResult;
type ListIteratee<T> = ListIterator<T, NotVoid> | IterateeShorthand<T>;
type ListIterateeCustom<T, TResult> = ListIterator<T, TResult> | IterateeShorthand<T>;
type MemoListIterator<T, TResult, TList> = (prev: TResult, curr: T, index: number, list: TList) => TResult;

const eachInArray = <T>(collection: T[] | null | undefined, iteratee: ArrayIterator<T, any>): T[] | null | undefined => {
	if (!collection) {
		return collection;
	}

	for (let i = 0; i < collection.length; i++) {
		if (false === iteratee(collection[i], i, collection)) {
			break;
		}
	}

	return collection;
};

const eachInObject = <T extends { [key: string]: unknown }>(
	collection: T | null | undefined,
	iteratee: ObjectIterator<T, any>
): T | null | undefined => {
	if (!collection) {
		return collection;
	}

	const keys = Object.keys(collection);
	for (let i = 0; i < keys.length; i++) {
		const key = keys[i];
		if (false === iteratee(collection[key] as T[keyof T], key, collection)) {
			break;
		}
	}
	return collection;
};

const filterArray = <T>(collection: List<T> | null | undefined, predicate: ListIterator<T, boolean>): T[] => {
	if (!collection) {
		return [];
	}

	const result: T[] = [];
	for (let i = 0; i < collection.length; i++) {
		if (false === predicate(collection[i], i, collection)) {
			continue;
		}
		result.push(collection[i]);
	}
	return result;
};

const filterObject = <T extends { [key: string]: unknown }>(
	collection: T | null | undefined,
	predicate: ObjectIterator<T, boolean>
): Array<T[keyof T]> => {
	if (!collection) {
		return [];
	}

	const res: Array<T[keyof T]> = [];
	const keys = Object.keys(collection);
	for (let i = 0; i < keys.length; i++) {
		const key = keys[i];
		const value = collection[key];
		if (true === predicate(value as T[keyof T], key, collection)) {
			res.push(value as T[keyof T]);
		}
	}
	return res;
};

const mapArray = <T, TResult>(collection: T[] | null | undefined, iteratee: ArrayIterator<T, TResult>): TResult[] => {
	if (!collection) {
		return [];
	}

	const result: TResult[] = [];
	for (let i = 0; i < collection.length; i++) {
		result.push(iteratee(collection[i], i, collection));
	}
	return result;
};

const mapObject = <T extends { [key: string]: unknown }, TResult>(
	collection: T | null | undefined,
	iteratee: ObjectIterator<T, TResult>
): TResult[] => {
	if (!collection) {
		return [];
	}

	const result: TResult[] = [];
	const keys = Object.keys(collection);
	for (let i = 0; i < keys.length; i++) {
		const key = keys[i];
		const value = collection[key];
		result.push(iteratee(value as T[keyof T], key, collection));
	}
	return result;
};

const reduceArray = <T, TResult>(
	collection: T[] | null | undefined,
	callback: MemoListIterator<T, TResult, T[]>,
	accumulator: TResult
): TResult => {
	if (!collection) {
		return accumulator;
	}

	for (let i = 0; i < collection.length; i++) {
		accumulator = callback(accumulator, collection[i], i, collection);
	}
	return accumulator;
};

const reduceObject = <T extends { [key: string]: unknown }, TResult>(
	collection: T | null | undefined,
	callback: MemoObjectIterator<T[keyof T], TResult, T>,
	accumulator: TResult
): TResult => {
	if (!collection) {
		return accumulator;
	}

	const keys = Object.keys(collection);

	for (let i = 0; i < keys.length; i++) {
		const key = keys[i];
		const value = collection[key];
		accumulator = callback(accumulator, value as T[keyof T], key, collection);
	}
	return accumulator;
};

function numberAsFixedString(value: number, digits: number, bounds: number[] = [], errString: string = "N/A"): string {
	if (!isFiniteNumber(value)) {
		return errString;
	}

	if (2 === bounds.length) {
		if (value < bounds[0] || value > bounds[1]) {
			return errString;
		}
	}

	return value.toFixed(digits);
}

const timestamp = (t?: number | Date): number => {
	if (undefined !== t) {
		if ("number" === typeof t) {
			return t;
		}
		if (t instanceof Date) {
			return t.valueOf();
		}
	}
	return Date.now();
};

class UTC {
	private value: number = 0;

	constructor(value?: number | Date) {
		this.value = timestamp(value);
	}

	// now is a static method that returns the current timestamp in UTC milliseconds.
	static now(): number {
		return Date.now();
	}

	// valueOf returns the current timestamp in UTC milliseconds.
	public valueOf(): number {
		return this.value;
	}

	// since returns the difference between the given timestamp and the current timestamp in milliseconds.
	public since(t: number | Date): number {
		const ts = timestamp(t);
		if (this.value > ts) {
			return this.value - ts;
		}

		return 0;
	}

	// until returns the difference between the current timestamp and the given timestamp in milliseconds.
	public until(t: number | Date): number {
		const ts = timestamp(t);
		if (ts > this.value) {
			return ts - this.value;
		}

		return 0;
	}

	// delta returns the absolute difference between the current timestamp and the given timestamp in milliseconds.
	public delta(t: number | Date): number {
		const ts = timestamp(t);
		if (this.value > ts) {
			return this.value - ts;
		}

		return ts - this.value;
	}

	// toDate converts the UTC object to a Date object in the local timezone.
	public toDate(): Date {
		return new Date(this.value);
	}
}

/** Provides a unit of time in milliseconds, corresponding to the named unit of time provided */
function unitsToMilliseconds(units: string): number {
	switch (units) {
		case "ms":
		case "msec":
		case "millisec":
		case "millisecond":
			return 1;
		case "s":
		case "sec":
		case "seconds":
			return 1000;
		case "m":
		case "min":
		case "minutes":
			return 60000;
		case "h":
		case "hour":
		case "hours":
			return 3600000;
		case "d":
		case "day":
		case "days":
			return 86400000;
		case "w":
		case "week":
		case "weeks":
			return 604800000;
		case "y":
		case "year":
		case "years":
			return 31536000000;
		default:
			return 1;
	}
}

function milliseconds(time: string | number | Date | UTC): number {
	const objType = typeof time;

	if ("string" === objType) {
		const temp = <string>time;
		let result = 0;

		let units = "";
		let numberString = "";

		for (let i = 0; i < temp.length; i++) {
			const code = temp.charCodeAt(i);

			// If the character is an ascii number, add it to the number string
			if ((code >= 48 && code <= 57) || 46 === code || 45 === code) {
				if (units !== "" && numberString !== "") {
					if (numberString !== "") {
						const multiplier = unitsToMilliseconds(units);

						const number = parseFloat(numberString);
						if (_.isFinite(number)) {
							result += number * multiplier;
						}
						numberString = "";
					}
					units = "";
				}

				numberString += temp[i];
				continue;
			}

			// If the character is an ascii letter, add it to the units string
			if (code >= 97 && code <= 122) {
				units += temp[i];
				continue;
			}
			if (code >= 65 && code <= 90) {
				units += temp[i].toLowerCase();
				continue;
			}
		}

		if (units !== "" && numberString !== "") {
			const multiplier = unitsToMilliseconds(units);

			const number = parseFloat(numberString);
			if (_.isFinite(number)) {
				result += number * multiplier;
			}
			units = "";
			numberString = "";
		}

		return _.round(result);
	}

	if ("number" === objType) {
		return _.round(<number>time);
	}

	if (time instanceof UTC) {
		return time.valueOf();
	}

	if (time instanceof Date) {
		return time.getTime();
	}

	throw new Error("Invalid input");
}

function millisecondsToString(time: string | number | Date | UTC): string {
	let ms = milliseconds(time);

	if (ms < 1) {
		return "0ms";
	}

	const years = Math.floor(ms / 31536000000);
	ms -= years * 31536000000;
	const weeks = Math.floor(ms / 604800000);
	ms -= weeks * 604800000;
	const days = Math.floor(ms / 86400000);
	ms -= days * 86400000;
	const hours = Math.floor(ms / 3600000);
	ms -= hours * 3600000;
	const minutes = Math.floor(ms / 60000);
	ms -= minutes * 60000;
	const seconds = Math.floor(ms / 1000);
	ms -= seconds * 1000;

	const result: string[] = [];
	if (years > 0) {
		result.push(`${years}y`);
	}

	if (weeks > 0) {
		result.push(`${weeks}w`);
	}

	if (days > 0) {
		result.push(`${days}d`);
	}

	if (hours > 0) {
		result.push(`${hours}h`);
	}

	if (minutes > 0) {
		result.push(`${minutes}m`);
	}

	if (seconds > 0) {
		result.push(`${seconds}s`);
	}

	if (ms > 0) {
		result.push(`${ms}ms`);
	}

	return result.join(" ");
}

export default {
	..._,
	asin: Math.asin,
	asyncEach,
	asyncForEach,
	atan2: Math.atan2,
	atan: Math.atan,
	ceil: Math.sqrt,
	closeTo,
	cos: Math.cos,
	defaultsTo: _.defaultTo, // Add an alias for this because it's annoying.
	defaultsToOneOf,
	errToString,
	filename,
	setParameter,
	getParameter,
	getParameterCaseInsensitive,
	Int8Array: {
		stringify: Int8ArrayStringify
	},
	Int16Array: {
		stringify: Int16ArrayStringify
	},
	Int32Array: {
		stringify: Int32ArrayStringify
	},
	lo_max: _.max,
	lo_min: _.min,
	mapKeysDeep,
	mapValuesDeep,
	max,
	min,
	sin: Math.sin,
	sqrt: Math.sqrt,
	strcmp,
	strstr,
	Uint8Array: {
		stringify: Uint8ArrayStringify
	},
	Uint16Array: {
		stringify: Uint16ArrayStringify
	},
	Uint32Array: {
		stringify: Uint32ArrayStringify
	},
	isFiniteNumber,
	isErrnoException,
	MACAddress,
	minOfEnum,
	maxOfEnum,
	shuffle,
	eachInArray,
	eachInObject,
	filterArray,
	filterObject,
	mapArray,
	mapObject,
	reduceArray,
	reduceObject,
	isArrayEqual,
	numberAsFixedString,
	unitsToMilliseconds,
	milliseconds,
	millisecondsToString,
	timestamp,
	UTC
};
