import re, collections

Tok = collections.namedtuple('Tok', 'type value')

class ImportSpec(object):
	__slots__ = ('name', 'path', 'comment_group')
	def __init__(self, name, path, comment_group):
		self.name = name
		self.path = path
		self.comment_group = comment_group

class PackageSpec(object):
	__slots__ = ('name', 'comment_group')
	def __init__(self, name, comment_group):
		self.name = name
		self.comment_group = comment_group

class ParserError(Exception):
	def __init__(self, msg):
		self.value = msg
	def __str__(self):
		return repr(self.value)

def parser_type_error(type, tok):
	return ParserError('expected: %s, got: %s' % (type, tok.type))

def token_iter(source):
	tok_spec = [
		('IDENT',   r'\w+'),
		('LPAREN',  r'\('),
		('RPAREN',  r'\)'),
		('STRING',  r'("([^"\\]|\\.)*"|`[^`]*`)'),
		('NEWLINE', r'\n'),
		('DOT',     r'\.'),
		('SEMI',    r';'),
		('COMMENT', r'//[^\n]*|/\*.*?\*/'),
		('SKIP',    r'[ \t\r]'),
	]

	tok_re = '|'.join('(?P<%s>%s)' % spec for spec in tok_spec)
	gettok = re.compile(tok_re, re.U | re.S).match

	lasttok = 'NEWLINE'

	pos = 0
	m = gettok(source, pos)
	while m is not None:
		tok = m.lastgroup
		pos = m.end()

		# automatic semicolon insertion
		if tok == 'NEWLINE':
			if lasttok in ['STRING', 'RPAREN', 'IDENT']:
				yield Tok('SEMI', ';')
				lasttok = 'SEMI'
			if lasttok == 'NEWLINE':
				yield Tok('EMPTYLINE', '')

		# here we need to intermix comments and newlines when they come
		# next to each other, because that will make EMPTYLINE tokens
		# correct
		if tok == 'COMMENT':
			if lasttok == 'NEWLINE':
				lasttok = tok
		elif tok != 'SKIP':
			lasttok = tok

		if tok not in ['SKIP', 'NEWLINE']:
			yield Tok(tok, m.group())

		m = gettok(source, pos)

class GoParser(object):
	def __init__(self, source):
		self.source = source + "\n\n"
		self.iter = token_iter(self.source)
		self.comment_group = []
		self.statements = []
		self.tok = None

	def next(self, skip_comments=True):
		self.tok = next(self.iter)
		if skip_comments:
			while self.tok.type in ['COMMENT', 'EMPTYLINE']:
				self.tok = next(self.iter)


	def expect(self, type, skip_comments=True):
		if self.tok.type != type:
			raise parser_type_error(type, self.tok)
		value = self.tok.value
		self.next(skip_comments)
		return value

	def expect_ident(self, ident):
		if self.tok.type != 'IDENT':
			raise parser_type_error('IDENT', self.tok)
		if self.tok.value != ident:
			raise ParserError('expected IDENT: %s, got: %s', ident, self.tok.value)
		value = self.tok.value
		self.next()
		return value

	def flush_comment_group(self):
		out = self.comment_group
		self.comment_group = []
		return out

	def comment(self, tok):
		if tok.type == 'EMPTYLINE':
			self.comment_group = []
			return

		if not self.comment_group:
			self.comment_group = [tok]
		else:
			self.comment_group.append(tok)

	def skip_comments(self):
		while self.tok.type in ['COMMENT', 'EMPTYLINE']:
			self.comment(self.tok)
			self.next(False)

	def parse_import_spec(self):
		name = None
		if self.tok.type == 'DOT':
			name = '.'
			self.next()
		elif self.tok.type == 'IDENT':
			name = self.tok.value
			self.next()

		path = self.expect('STRING')
		return ImportSpec(name, path[1:-1], [])

	def parse_import_group(self):
		specs = []
		self.expect('LPAREN', False)
		while 1:
			self.skip_comments()
			if self.tok.type in ['DOT', 'IDENT', 'STRING']:
				spec = self.parse_import_spec()
				spec.comment_group = self.flush_comment_group()
				specs.append(spec)
				if self.tok.type == 'SEMI':
					self.next(False)
			elif self.tok.type == 'RPAREN':
				self.next()
				return specs
			else:
				raise parser_type_error('RPARENT, DOT, IDENT or STRING', self.tok)

	def parse_import_spec_or_group(self):
		if self.tok.type == 'LPAREN':
			return self.parse_import_group()
		elif self.tok.type in ['DOT', 'IDENT', 'STRING']:
			return [self.parse_import_spec()]
		raise parser_type_error('LPAREN, DOT, IDENT or STRING', self.tok)

	def parse_import_statement(self):
		cg = self.flush_comment_group()
		self.expect_ident('import')
		specs = self.parse_import_spec_or_group()
		if len(specs) == 1 and not specs[0].comment_group:
			specs[0].comment_group = cg
		for s in specs:
			self.statements.append(s)

	def parse(self):
		try:
			self.next(False)
			self.skip_comments()

			# package clause
			self.expect_ident('package')
			name = self.expect('IDENT')
			self.expect('SEMI', False)

			self.statements.append(
				PackageSpec(name, self.flush_comment_group())
			)

			# import clauses
			self.skip_comments()
			while self.tok.type == 'IDENT' and self.tok.value == 'import':
				self.parse_import_statement()
				self.expect('SEMI', False)
				self.skip_comments()
		except StopIteration:
			pass

# these imports are down there, because I wanted to emphasize that the parser
# itself does not depend on waf at all
from waflib import Utils
from waflib.Errors import WafError

def find_local(path, dir):
	node = dir.find_resource(path + '.a')
	if node:
		return node

	node = dir.find_resource(path + '.' + dir.ctx.env.GOCHAR)
	if node:
		return node

	return None

def find_global(path, nodepaths):
	for dir in nodepaths:
		node = find_local(path, dir)
		if node:
			return node

	return None

def scan(task):
	try:
		incn = task.generator.includes_nodes
	except AttributeError:
		raise WafError('%r is missing a feature such as "go" or "includes": ' % task.generator)

	# filter out nodes that are not in the project directory, we don't care
	# about these
	nodepaths = [x for x in incn if x.is_child_of(x.ctx.bldnode) or x.is_child_of(x.ctx.srcnode)]
	bldnode = task.generator.bld.bldnode

	deps = []
	for input in task.inputs:
		file = Utils.readf(input.abspath())
		try:
			gp = GoParser(file)
			gp.parse()
		except ParserError:
			pass

		for s in gp.statements:
			if not isinstance(s, ImportSpec):
				continue

			# TODO: global paths should be treated as local too, but
			# no one uses them?
			if s.path.startswith("./") or s.path.startswith("../"):
				node = find_local(s.path, bldnode)
			else:
				node = find_global(s.path, nodepaths)

			if node:
				deps.append(node)

	return (deps, None)

################################################################################
# REMOVE THE CODE BELOW BEFORE SUBMITTING
################################################################################

def test():
	def panic(n, s):
		raise Exception("Panic in test %d: %s" % (n, s))

	def deep_compare_spec(n, e, s):
		if s.__class__ != e.__class__:
			panic(n, "Type mismatch, expected: %s, got: %s" % (e.__class__.__name__, s.__class__.__name__))
		cls = s.__class__.__name__

		if s.name != e.name:
			panic(n, "%s name mismatch, expected: %s, got: %s" % (cls, e.name, s.name))
		if len(s.comment_group) != len(e.comment_group):
			panic(n, "%s comment group length mismatch, expected: %d, got: %d" % (cls, len(e.comment_group), len(s.comment_group)))
		for sc, ec in zip(s.comment_group, e.comment_group):
			if sc.value != ec:
				panic(n, "%s comment mismatch, expected: %s, got: %s" % (cls, ec, sc.value))

		if isinstance(s, ImportSpec):
			if s.path != e.path:
				panic(n, "ImportSpec path mismatch, expected: %s, got: %s" % (e.path, s.path))

	tests = []
	expectations = []
	#------------------------------------------------------------------
	tests.append("""
package myLife146_77; import (`X`;`Y`)
	""")
	expectations.append([
		PackageSpec('myLife146_77', []),
		ImportSpec(None, 'X', []),
		ImportSpec(None, 'Y', []),
	])
	#------------------------------------------------------------------
	tests.append("""
// package comment
package main

// import comment
import "main"
	""")
	expectations.append([
		PackageSpec('main', ['// package comment']),
		ImportSpec(None, 'main', ['// import comment']),
	])
	#------------------------------------------------------------------
	tests.append("""
package main

// import comment 1
import (
	// import comment 2
	"test"
)
	""")
	expectations.append([
		PackageSpec('main', []),
		ImportSpec(None, 'test', ['// import comment 2']),
	])
	#------------------------------------------------------------------
	tests.append("""
/* nowhere comment */

/*pkg comment*/package main

// import comment
import (
	"test1"
	"test2"
)

import . "dot"
import /* comment here too */ _ /* comment in the middle */"ignored" // oops
import localname `localname `

import (
	// x
	/* y */
	"A"
	// z
	"B"
)
	""")
	expectations.append([
		PackageSpec('main', ['/*pkg comment*/']),
		ImportSpec(None, 'test1', []),
		ImportSpec(None, 'test2', []),
		ImportSpec('.', 'dot', []),
		ImportSpec('_', 'ignored', []),
		ImportSpec('localname', 'localname ', []),
		ImportSpec(None, 'A', ['// x', '/* y */']),
		ImportSpec(None, 'B', ['// z']),
	])
	#------------------------------------------------------------------
	tests.append("""/*test*/package x""")
	expectations.append([
		PackageSpec('x', ['/*test*/']),
	])
	tests.append("""
package x /* wtf */
import "X"
""")
	expectations.append([
		PackageSpec('x', []),
		ImportSpec(None, 'X', []),
	])
	#------------------------------------------------------------------
	for n, test in enumerate(tests):
		ee = expectations[n]
		gp = GoParser(test)
		gp.parse()
		if len(gp.statements) != len(ee):
			panic(n, "Unexpected number of statements, expected: %d, got: %d" % (len(ee), len(gp.statements)))
		for s, e in zip(gp.statements, ee):
			deep_compare_spec(n, e, s)

if __name__ == '__main__':
	test()
