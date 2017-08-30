#!/usr/bin/perl

use strict;

my %frmVals = ();
my $ret = &ReadForm(\%frmVals);

#print "Content-type: plain/text\n\n";

if ($frmVals{'file'}) {
	my $file = $frmVals{'file'};
	
	open(INPUT, "<:utf8", "/tmp/$file") || die "Content-type: text/html\n\nCannot open /tmp/$file!: $!";
		print "Content-Type:application/x-download\n";
		print "Content-Disposition:attachment;filename=$file\n\n";

		while (<INPUT>) {
			print $_;
		}
	close(INPUT);
}


#==============================================================
# subroutine "ReadForm" parses the data string that comes in
# from a client web browser by way of Apache and stores it in 
# a hash called %variable, where the keys are the names of the 
# form items, and the values are their respective contents

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


