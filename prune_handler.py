#!/usr/bin/python2.7
# -*- coding: utf-8 -*-

from google.appengine.ext.webapp import template
from google.appengine.ext.webapp.util import run_wsgi_app
from google.appengine.api import urlfetch


import os
import webapp2
import httplib2
import urllib
import logging
import math
import numpy
import StringIO
import pprint

os.environ["MATPLOTLIBDATA"] = os.getcwdu()
os.environ["MPLCONFIGDIR"] = os.getcwdu()
import subprocess
def no_popen(*args, **kwargs): raise OSError("forbjudet")
subprocess.Popen = no_popen
subprocess.PIPE = None
subprocess.STDOUT = None
#logging.warn("E: %s" % pprint.pformat(os.environ))
#try:
#    import numpy, matplotlib, matplotlib.pyplot as plt
#except:
#    logging.exception("trouble")

from Bio import Phylo

from google.appengine.api import urlfetch
from google.appengine.api import taskqueue


import json
#from oauth2client.appengine import AppAssertionCredentials

urlfetch.set_default_fetch_deadline(60)


class PruneHandler(webapp2.RequestHandler):
    def get(self):
        logging.info('issa get')
        self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        self.response.headers['Content-Type'] = 'text/csv'

    def post(self):
        try:
            logging.info('loading tree')
            self.response.headers.add_header("Access-Control-Allow-Origin", "*")
            self.response.headers['Content-Type'] = 'application/tre'
            tree = Phylo.read('trees/xaa','newick')
            logging.info('pruning tree')



            treenum = 5 #int(self.request.get('treenum',10))
            names = str(self.request.get('species','').replace(' ','_')).split('\n')
            logging.info(treenum)
            pruned_trees = []

            for i in range(treenum):

                for name in names:
                	tree.prune(name)

                logging.info('saving tree %s' % i)
                Phylo.write(tree, self.response.out,'newick')
                
                self.response.out.write(tree)
                tree = Phylo.read('trees/xaa','newick')

            
           
            (pruned_trees)
        except Exception as e:
            logging.info(e)

application = webapp2.WSGIApplication(
    [ ('/api/prune', PruneHandler)], debug=True)
