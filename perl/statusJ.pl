#!/usr/bin/perl

use strict;
use JSON;

my %frmVals = ();
my $ret = &ReadForm(\%frmVals);

my $pid = $frmVals{'pid'};
my $callback = $frmVals{'callback'};

if ($callback) { 
    print "Access-Control-Allow-Origin: *\n";
    print "Access-Control-Allow-Methods: GET\n"; 
    print "Content-type: application/javascript\n\n";
} else { 
    # Header for access via browser, curl, etc. 
    print "Access-Control-Allow-Origin: *\n";
    print "Content-type: application/json\n\n"; 
}

my %json;
if ($pid =~ m/^[0-9]+$/) {
	my $trees_done = `grep -c -P '^\tTREE ' /tmp/$pid.tre`;
	chop($trees_done);
	push @{ $json{results} }, { "trees_done" => $trees_done } ;
} else {
	push @{ $json{results} }, { "error" => "no PID" } ;
}

my $json = to_json(\%json, {pretty=>'1'}); 
if ($callback) { 
	print $callback . '(' . $json . ');'; 
} else { 
	print $json,"\n"; 
}


#==============================================================
sub ReadForm {
	my ($varRef) = @_;
	my ($i, $key, $val, $variable, @list);
	
	if (&MethGet) {
		$variable = $ENV{'QUERY_STRING'};
	} elsif (&MethPost) {
		read(STDIN, $variable, $ENV{'CONTENT_LENGTH'});
	}
	
	@list = split(/[&;]/,$variable);
	
	foreach $i (0 .. $#list) {
		$list[$i] =~ s/\+/ /g;
		($key, $val) = split(/=/,$list[$i],2);
		$key =~ s/%(..)/pack("c",hex($1))/ge;
		$val =~ s/%(..)/pack("c",hex($1))/ge;
		$$varRef{$key} .= "\0" if (defined($$varRef{$key}));
		$$varRef{$key} .= $val;
	}
	return scalar($#list + 1);
}
sub MethGet { return ($ENV{'REQUEST_METHOD'} eq "GET") };
sub MethPost { return ($ENV{'REQUEST_METHOD'} eq "POST") };

